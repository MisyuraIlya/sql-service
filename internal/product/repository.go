package product

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sql-service/pkg/db"
)

type MyNullFloat64 struct{ sql.NullFloat64 }

func (n MyNullFloat64) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Float64)
	}
	return []byte("null"), nil
}

type MyNullString struct{ sql.NullString }

func (s MyNullString) MarshalJSON() ([]byte, error) {
	if s.Valid {
		return json.Marshal(s.String)
	}
	return []byte("null"), nil
}

type ProductRepository struct{ Db *db.Db }

func NewProductRepository(db *db.Db) *ProductRepository { return &ProductRepository{Db: db} }

func (r *ProductRepository) GetProducts(dto *ProductsDto) ([]Product, error) {
	if len(dto.Skus) == 0 {
		return nil, fmt.Errorf("sku list cannot be empty")
	}

	unionParts := make([]string, 0, len(dto.Skus))
	args := []any{
		sql.Named("cardCode", dto.CardCode),
		sql.Named("userExtId", dto.CardCode),
		sql.Named("asOfDate", dto.Date),
		sql.Named("warehouse", dto.Warehouse),
	}

	for i, sku := range dto.Skus {
		name := fmt.Sprintf("sku%d", i)
		if i == 0 {
			unionParts = append(unionParts, fmt.Sprintf("SELECT @%s AS sku", name))
		} else {
			unionParts = append(unionParts, fmt.Sprintf("UNION ALL SELECT @%s", name))
		}
		args = append(args, sql.Named(name, sku))
	}
	skuUnion := strings.Join(unionParts, "\n        ")

	query := fmt.Sprintf(`
WITH SkuList AS (
        %s
),
Cust AS (
    SELECT TOP 1 T5.ListNum
    FROM OCRD AS T5 WITH (NOLOCK)
    WHERE T5.CardCode = @cardCode
),
CustGroup AS (
    SELECT TOP 1 GroupCode
    FROM OCRD WITH (NOLOCK)
    WHERE CardCode = @cardCode
),
BasePrice AS (
    SELECT
        OITM.ItemCode,
        OITM.FirmCode,
        ITM1.Price    AS PriceListPrice,
        ITM1.Currency AS Currency
    FROM OITM WITH (NOLOCK)
    INNER JOIN ITM1 WITH (NOLOCK)
        ON OITM.ItemCode = ITM1.ItemCode
    CROSS JOIN Cust
    WHERE ITM1.PriceList = Cust.ListNum
      AND OITM.ItemCode IN (SELECT sku FROM SkuList)
),
SpecialPrice AS (
    SELECT
        P.ItemCode,
        P.ListNum,
        P.Price    AS OSPPPrice,
        P.Discount AS OSPPDiscount
    FROM OSPP AS P WITH (NOLOCK)
    WHERE P.CardCode = @cardCode
      AND P.Valid = 'Y'
      AND (
            (P.ValidFrom IS NULL AND P.ValidTo IS NULL)
         OR (P.ValidFrom <= @asOfDate AND (P.ValidTo IS NULL OR P.ValidTo >= @asOfDate))
      )
),
AllDiscountRules AS (
    SELECT
        BP.ItemCode,
        E.Type                   AS RuleType,
        E1.ObjType,
        E1.ObjKey,
        E1.Discount              AS DiscountPct,
        CASE E1.ObjType
            WHEN '4'  THEN N'Discount group (item)'
            WHEN '43' THEN N'Discount group (manufacturer)'
            WHEN '52' THEN N'Discount group (item group)'
        END                      AS RuleSource
    FROM BasePrice AS BP
    CROSS JOIN CustGroup
    INNER JOIN OEDG AS E WITH (NOLOCK)
        ON E.ValidFor = 'Y'
       AND (
             (E.Type = 'S' AND E.ObjCode = @cardCode)
          OR (E.Type = 'C' AND E.ObjCode = CONVERT(NVARCHAR, CustGroup.GroupCode))
          OR (E.ObjType = '-1' AND E.ObjCode = '0') -- global rule
          )
    INNER JOIN EDG1 AS E1 WITH (NOLOCK)
        ON E1.AbsEntry = E.AbsEntry
       AND E1.ObjType IN ('4','43','52')
    LEFT JOIN OITM WITH (NOLOCK)
        ON OITM.ItemCode = BP.ItemCode
    WHERE
          (E1.ObjType = '4'  AND E1.ObjKey = BP.ItemCode)
       OR (E1.ObjType = '43' AND TRY_CAST(E1.ObjKey AS INT) = BP.FirmCode)
       OR (E1.ObjType = '52' AND TRY_CAST(E1.ObjKey AS INT) = OITM.ItmsGrpCod)
),
BestDiscountPerItem AS (
    SELECT ItemCode,
           DiscountPct,
           RuleSource
    FROM (
        SELECT
            R.ItemCode,
            R.DiscountPct,
            R.RuleSource,
            ROW_NUMBER() OVER (
                PARTITION BY R.ItemCode
                ORDER BY
                    CASE R.ObjType
                        WHEN '4'  THEN 1
                        WHEN '43' THEN 2
                        WHEN '52' THEN 3
                        ELSE 4
                    END,
                    R.DiscountPct DESC
            ) AS rn
        FROM AllDiscountRules AS R
    ) X
    WHERE rn = 1
),
PromoDiscount AS (
    SELECT
        I.ItemCode,
        E1.Discount AS PromoDiscount
    FROM OEDG AS E WITH (NOLOCK)
    INNER JOIN EDG1 AS E1 WITH (NOLOCK)
        ON E1.AbsEntry = E.AbsEntry
       AND E1.ObjType = '4'
    INNER JOIN OITM AS I WITH (NOLOCK)
        ON I.ItemCode = E1.ObjKey
    WHERE E.ValidFor = 'Y'
      AND E.Type = 'A'
),
Stock AS (
    SELECT
        W.ItemCode,
        W.WhsCode    AS warehouseCode,
        W.OnHand     AS stock,
        W.OnOrder    AS onOrder,
        W.IsCommited AS commited
    FROM OITW AS W WITH (NOLOCK)
    WHERE W.WhsCode = @warehouse
)
SELECT
    BP.ItemCode                                                      AS sku,
    @cardCode                                                        AS CardCode,
    CAST((SELECT ListNum FROM Cust) AS DECIMAL(19,4))                AS PriceList,
    BP.Currency,
    CAST(BP.PriceListPrice AS DECIMAL(19,4))                         AS PriceListPrice,
    CAST(SP.OSPPPrice AS DECIMAL(19,4))                              AS OSPPPrice,
    CAST(SP.OSPPDiscount AS DECIMAL(19,4))                           AS OSPPDiscount,
    CAST(BD.DiscountPct AS DECIMAL(19,4))                            AS BPGroupDiscount,
    CAST(NULL AS NVARCHAR(255))                                      AS ManufacturerName,
    CAST(NULL AS DECIMAL(19,4))                                      AS ManufacturerDiscount,
    CAST(PD.PromoDiscount AS DECIMAL(19,4))                          AS PromoDiscount,
    ISNULL(S.warehouseCode, '')                                      AS warehouseCode,
    CAST(S.stock AS DECIMAL(19,4))                                   AS stock,
    CAST(S.onOrder AS DECIMAL(19,4))                                 AS onOrder,
    CAST(S.commited AS DECIMAL(19,4))                                AS commited,
    CASE
        WHEN SP.OSPPPrice IS NOT NULL AND SP.OSPPPrice > 0 THEN N'OSPP explicit price'
        WHEN SP.OSPPDiscount IS NOT NULL THEN N'OSPP discount'
        WHEN BD.DiscountPct IS NOT NULL THEN BD.RuleSource
        WHEN PD.PromoDiscount IS NOT NULL THEN N'Promo (EDG Type A)'
        ELSE N'Base price list'
    END                                                              AS PriceSource,
    CAST(
        CASE
            WHEN SP.OSPPPrice IS NOT NULL AND SP.OSPPPrice > 0
                THEN SP.OSPPPrice
            WHEN SP.OSPPDiscount IS NOT NULL
                THEN BP.PriceListPrice * (100.0 - SP.OSPPDiscount) / 100.0
            WHEN BD.DiscountPct IS NOT NULL
                THEN BP.PriceListPrice * (100.0 - BD.DiscountPct) / 100.0
            WHEN PD.PromoDiscount IS NOT NULL
                THEN BP.PriceListPrice * (100.0 - PD.PromoDiscount) / 100.0
            ELSE BP.PriceListPrice
        END
        AS DECIMAL(19,4)
    )                                                                AS FinalPrice
FROM BasePrice AS BP
LEFT JOIN SpecialPrice AS SP
       ON SP.ItemCode = BP.ItemCode
      AND (SP.ListNum IS NULL OR SP.ListNum = (SELECT ListNum FROM Cust))
LEFT JOIN BestDiscountPerItem AS BD
       ON BD.ItemCode = BP.ItemCode
LEFT JOIN PromoDiscount AS PD
       ON PD.ItemCode = BP.ItemCode
LEFT JOIN Stock AS S
       ON S.ItemCode = BP.ItemCode
ORDER BY BP.ItemCode;
`, skuUnion)

	rows, err := r.Db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.SKU,
			&p.CardCode,
			&p.PriceList,
			&p.Currency,
			&p.PriceListPrice,
			&p.OSPPPrice,
			&p.OSPPDiscount,
			&p.BPGroupDiscount,
			&p.ManufacturerName,
			&p.ManufacturerDiscount,
			&p.PromoDiscount,
			&p.WarehouseCode,
			&p.Stock,
			&p.OnOrder,
			&p.Commited,
			&p.PriceSource,
			&p.FinalPrice,
		); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *ProductRepository) GeTreeProducts(dto *ProductsTreeDto) ([]BomHeaderDTO, error) {
	if len(dto.Skus) == 0 {
		return nil, fmt.Errorf("sku list cannot be empty")
	}

	// Build UNION list of SKUs as parameters
	unionParts := make([]string, 0, len(dto.Skus))
	args := make([]any, 0, len(dto.Skus))
	for i, sku := range dto.Skus {
		name := fmt.Sprintf("sku%d", i)
		if i == 0 {
			unionParts = append(unionParts, fmt.Sprintf("SELECT @%s AS sku", name))
		} else {
			unionParts = append(unionParts, fmt.Sprintf("UNION ALL SELECT @%s", name))
		}
		args = append(args, sql.Named(name, sku))
	}
	parentSkuUnion := strings.Join(unionParts, "\n        ")

	// 1) Headers
	headersSQL := fmt.Sprintf(`
WITH ParentSkuList AS (
        %s
)
SELECT
    Code, TreeType, PriceList, Qauntity, CreateDate, UpdateDate, Transfered,
    DataSource, UserSign, SCNCounter, DispCurr, ToWH, Object, LogInstac,
    UserSign2, OcrCode, HideComp, OcrCode2, OcrCode3, OcrCode4, OcrCode5,
    UpdateTime, Project, PlAvgSize, Name, CreateTS, UpdateTS, AtcEntry,
    Attachment, U_UPI_Ignore, U_UPI_ProductionTree, U_XIS_Comments
FROM OITT WITH (NOLOCK)
WHERE Code IN (SELECT sku FROM ParentSkuList)
`, parentSkuUnion)

	hRows, err := r.Db.Query(headersSQL, args...)
	if err != nil {
		return nil, err
	}
	defer hRows.Close()

	headersByCode := make(map[string]*BomHeaderDTO, len(dto.Skus))
	for hRows.Next() {
		// Declare pointer variables for scanning
		var (
			code, treeType                                string
			priceList, userSign, scnCounter, logInstac    *int64
			quantity, plAvgSize                           *float64
			createDate, updateDate                        *time.Time
			transfered, dataSource, dispCurr, toWH        *string
			object, ocrCode, hideComp, ocrCode2, ocrCode3 *string
			ocrCode4, ocrCode5, project, name             *string
			userSign2, atcEntry, attachment               *int64
			updateTime, createTS, updateTS                *int64 // int HHMMSS
			uUPIIgnore, uUPIProductionTree, uXISComments  *string
		)

		if err := hRows.Scan(
			&code, &treeType, &priceList, &quantity, &createDate, &updateDate, &transfered,
			&dataSource, &userSign, &scnCounter, &dispCurr, &toWH, &object, &logInstac,
			&userSign2, &ocrCode, &hideComp, &ocrCode2, &ocrCode3, &ocrCode4, &ocrCode5,
			&updateTime, &project, &plAvgSize, &name, &createTS, &updateTS, &atcEntry,
			&attachment, &uUPIIgnore, &uUPIProductionTree, &uXISComments,
		); err != nil {
			return nil, err
		}

		headersByCode[code] = &BomHeaderDTO{
			Code:                 code,
			TreeType:             treeType,
			PriceList:            priceList,
			Quantity:             quantity,
			CreateDate:           createDate,
			UpdateDate:           updateDate,
			Transfered:           transfered,
			DataSource:           dataSource,
			UserSign:             userSign,
			SCNCounter:           scnCounter,
			DispCurr:             dispCurr,
			ToWH:                 toWH,
			Object:               object,
			LogInstac:            logInstac,
			UserSign2:            userSign2,
			OcrCode:              ocrCode,
			HideComp:             hideComp,
			OcrCode2:             ocrCode2,
			OcrCode3:             ocrCode3,
			OcrCode4:             ocrCode4,
			OcrCode5:             ocrCode5,
			UpdateTime:           updateTime,
			Project:              project,
			PlAvgSize:            plAvgSize,
			Name:                 name,
			CreateTS:             createTS,
			UpdateTS:             updateTS,
			AtcEntry:             atcEntry,
			Attachment:           attachment,
			U_UPI_Ignore:         uUPIIgnore,
			U_UPI_ProductionTree: uUPIProductionTree,
			U_XIS_Comments:       uXISComments,
			Lines:                make([]BomLineDTO, 0, 8),
		}
	}
	if err := hRows.Err(); err != nil {
		return nil, err
	}
	if len(headersByCode) == 0 {
		return []BomHeaderDTO{}, nil
	}

	// 2) Lines
	linesSQL := fmt.Sprintf(`
WITH ParentSkuList AS (
        %s
)
SELECT
    Father, ChildNum, VisOrder, Code, Quantity, Warehouse, Price, Currency,
    PriceList, OrigPrice, OrigCurr, IssueMthd, Uom, Comment, LogInstanc,
    Object, OcrCode, OcrCode2, OcrCode3, OcrCode4, OcrCode5, PrncpInput,
    Project, Type, WipActCode, AddQuantit, LineText, StageId, ItemName,
    U_UPI_BaseEl, U_IsVisibleOnWebshop, U_InvCalc
FROM ITT1 WITH (NOLOCK)
WHERE Father IN (SELECT sku FROM ParentSkuList)
ORDER BY Father, VisOrder, ChildNum, Code
`, parentSkuUnion)

	lRows, err := r.Db.Query(linesSQL, args...)
	if err != nil {
		return nil, err
	}
	defer lRows.Close()

	for lRows.Next() {
		var (
			father, code                                         string
			childNum, visOrder, priceList, logInstanc, stageId   *int64
			quantity, price, origPrice, addQuantit               *float64
			warehouse, currency, origCurr, issueMthd, uom        *string
			comment, object, ocrCode, ocrCode2, ocrCode3         *string
			ocrCode4, ocrCode5, prncpInput, project, typ, wipAct *string
			lineText, itemName, uUPIBaseEl, uIsVisibleOnWebshop  *string
			uInvCalc                                             *string
		)

		if err := lRows.Scan(
			&father, &childNum, &visOrder, &code, &quantity, &warehouse, &price, &currency,
			&priceList, &origPrice, &origCurr, &issueMthd, &uom, &comment, &logInstanc,
			&object, &ocrCode, &ocrCode2, &ocrCode3, &ocrCode4, &ocrCode5, &prncpInput,
			&project, &typ, &wipAct, &addQuantit, &lineText, &stageId, &itemName,
			&uUPIBaseEl, &uIsVisibleOnWebshop, &uInvCalc,
		); err != nil {
			return nil, err
		}

		if h := headersByCode[father]; h != nil {
			h.Lines = append(h.Lines, BomLineDTO{
				Father:               father,
				ChildNum:             childNum,
				VisOrder:             visOrder,
				Code:                 code,
				Quantity:             quantity,
				Warehouse:            warehouse,
				Price:                price,
				Currency:             currency,
				PriceList:            priceList,
				OrigPrice:            origPrice,
				OrigCurr:             origCurr,
				IssueMthd:            issueMthd,
				Uom:                  uom,
				Comment:              comment,
				LogInstanc:           logInstanc,
				Object:               object,
				OcrCode:              ocrCode,
				OcrCode2:             ocrCode2,
				OcrCode3:             ocrCode3,
				OcrCode4:             ocrCode4,
				OcrCode5:             ocrCode5,
				PrncpInput:           prncpInput,
				Project:              project,
				Type:                 typ,
				WipActCode:           wipAct,
				AddQuantit:           addQuantit,
				LineText:             lineText,
				StageId:              stageId,
				ItemName:             itemName,
				U_UPI_BaseEl:         uUPIBaseEl,
				U_IsVisibleOnWebshop: uIsVisibleOnWebshop,
				U_InvCalc:            uInvCalc,
			})
		}
	}
	if err := lRows.Err(); err != nil {
		return nil, err
	}

	// 3) Result in input order
	result := make([]BomHeaderDTO, 0, len(headersByCode))
	seen := map[string]bool{}
	for _, sku := range dto.Skus {
		if h := headersByCode[sku]; h != nil && !seen[sku] {
			result = append(result, *h)
			seen[sku] = true
		}
	}
	for code, h := range headersByCode {
		if !seen[code] {
			result = append(result, *h)
		}
	}
	return result, nil
}
