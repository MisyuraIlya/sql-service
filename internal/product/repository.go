package product

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
	totalStart := time.Now()
	log.Printf("GetProducts: start, skus=%d, cardCode=%s, warehouse=%s, date=%s",
		len(dto.Skus), dto.CardCode, dto.Warehouse, dto.Date)

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
            (P.ValidFrom IS NULL OR P.ValidFrom <= @asOfDate)
        AND (P.ValidTo   IS NULL OR P.ValidTo   >= @asOfDate)
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
        ON (
             (E.Type = 'S' AND E.ObjCode = @cardCode)
          OR (E.Type = 'C' AND E.ObjCode = CONVERT(NVARCHAR, CustGroup.GroupCode))
          OR (E.ObjType = '-1' AND E.ObjCode = '0') -- global rule
          )
       AND (
            -- No date restriction → always valid
            E.ValidFor = 'N'
            OR
            -- Has validity period → must match @asOfDate
            (
                E.ValidFor = 'Y'
                AND (
                        (E.ValidForm IS NULL OR E.ValidForm <= @asOfDate)
                    AND (E.ValidTo   IS NULL OR E.ValidTo   >= @asOfDate)
                )
            )
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
           RuleSource,
           RuleType
    FROM (
        SELECT
            R.ItemCode,
            R.DiscountPct,
            R.RuleSource,
            R.RuleType,
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
    WHERE E.Type = 'A'
      AND (
            E.ValidFor = 'N'
         OR (
                E.ValidFor = 'Y'
            AND (
                    (E.ValidForm IS NULL OR E.ValidForm <= @asOfDate)
                AND (E.ValidTo   IS NULL OR E.ValidTo   >= @asOfDate)
            )
         )
      )
),

-- === BOM / stock logic (MIN stock of children) ===
TreeParents AS (
    SELECT S.sku AS ParentCode
    FROM SkuList AS S
    INNER JOIN OITT AS H WITH (NOLOCK)
        ON H.Code = S.sku
       AND H.TreeType = 'S'
),
ParentChildren AS (
    SELECT H.Code AS ParentCode,
           L.Code AS ChildCode
    FROM OITT AS H WITH (NOLOCK)
    INNER JOIN ITT1 AS L WITH (NOLOCK)
        ON L.Father = H.Code
    WHERE H.TreeType = 'S'
      AND H.Code IN (SELECT sku FROM SkuList)
),
AllItemsForStock AS (
    -- Non-tree parents: use own stock
    SELECT S.sku AS ParentCode,
           S.sku AS ItemCodeToCheck
    FROM SkuList AS S
    WHERE S.sku NOT IN (SELECT ParentCode FROM TreeParents)

    UNION ALL

    -- Tree parents: use children
    SELECT C.ParentCode,
           C.ChildCode AS ItemCodeToCheck
    FROM ParentChildren AS C
),
StockRaw AS (
    SELECT
        W.ItemCode,
        W.WhsCode,
        W.OnHand,
        W.OnOrder,
        W.IsCommited
    FROM OITW AS W WITH (NOLOCK)
    WHERE W.WhsCode = @warehouse
      AND W.ItemCode IN (SELECT ItemCodeToCheck FROM AllItemsForStock)
),
StockPerParentRows AS (
    SELECT
        A.ParentCode,
        S.ItemCode,
        S.WhsCode,
        S.OnHand,
        S.OnOrder,
        S.IsCommited,
        ROW_NUMBER() OVER (
            PARTITION BY A.ParentCode
            ORDER BY
                CASE WHEN S.OnHand IS NULL THEN 1 ELSE 0 END,
                S.OnHand ASC
        ) AS rn
    FROM AllItemsForStock AS A
    LEFT JOIN StockRaw AS S
      ON S.ItemCode = A.ItemCodeToCheck
),
Stock AS (
    SELECT
        SPR.ParentCode AS ItemCode,
        SPR.WhsCode    AS warehouseCode,
        SPR.OnHand     AS stock,
        SPR.OnOrder    AS onOrder,
        SPR.IsCommited AS commited
    FROM StockPerParentRows AS SPR
    WHERE SPR.rn = 1
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
    CAST(BD.RuleType AS NVARCHAR(1))                                 AS BPGroupDiscountType,
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
ORDER BY BP.ItemCode
OPTION (RECOMPILE);
`, skuUnion)

	log.Println("====== GetProducts SQL ======")
	log.Println(query)
	log.Println("====== GetProducts ARGS =====")
	for _, a := range args {
		if na, ok := a.(sql.NamedArg); ok {
			log.Printf("@%s = %v\n", na.Name, na.Value)
		} else {
			log.Printf("%T: %v\n", a, a)
		}
	}
	log.Println("====== END GetProducts DUMP =====")

	queryStart := time.Now()
	rows, err := r.Db.Query(query, args...)
	if err != nil {
		log.Printf("GetProducts: Query() error after %s: %v", time.Since(queryStart), err)
		return nil, err
	}
	log.Printf("GetProducts: Query() took %s", time.Since(queryStart))
	defer rows.Close()

	scanStart := time.Now()
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
			&p.BPGroupDiscountType,
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
			log.Printf("GetProducts: scan error after %s: %v", time.Since(scanStart), err)
			return nil, err
		}
		products = append(products, p)
	}
	log.Printf("GetProducts: scan loop took %s, rows=%d", time.Since(scanStart), len(products))

	if err := rows.Err(); err != nil {
		log.Printf("GetProducts: rows.Err(): %v", err)
		return nil, err
	}

	log.Printf("GetProducts: DONE total=%s, rows=%d", time.Since(totalStart), len(products))
	return products, nil
}

func (r *ProductRepository) GeTreeProducts(dto *ProductSkusDto) ([]BomHeaderDTO, error) {
	totalStart := time.Now()
	log.Printf("GeTreeProducts: start, skus=%d", len(dto.Skus))

	if len(dto.Skus) == 0 {
		return nil, fmt.Errorf("sku list cannot be empty")
	}

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

	hQueryStart := time.Now()
	hRows, err := r.Db.Query(headersSQL, args...)
	if err != nil {
		log.Printf("GeTreeProducts: headers Query() error after %s: %v", time.Since(hQueryStart), err)
		return nil, err
	}
	log.Printf("GeTreeProducts: headers Query() took %s", time.Since(hQueryStart))
	defer hRows.Close()

	headersByCode := make(map[string]*BomHeaderDTO, len(dto.Skus))
	hScanStart := time.Now()
	for hRows.Next() {
		var (
			code, treeType                                string
			priceList, userSign, scnCounter, logInstac    *int64
			quantity, plAvgSize                           *float64
			createDate, updateDate                        *time.Time
			transfered, dataSource, dispCurr, toWH        *string
			object, ocrCode, hideComp, ocrCode2, ocrCode3 *string
			ocrCode4, ocrCode5, project, name             *string
			userSign2, atcEntry, attachment               *int64
			updateTime, createTS, updateTS                *int64
			uUPIIgnore, uUPIProductionTree, uXISComments  *string
		)

		if err := hRows.Scan(
			&code, &treeType, &priceList, &quantity, &createDate, &updateDate, &transfered,
			&dataSource, &userSign, &scnCounter, &dispCurr, &toWH, &object, &logInstac,
			&userSign2, &ocrCode, &hideComp, &ocrCode2, &ocrCode3, &ocrCode4, &ocrCode5,
			&updateTime, &project, &plAvgSize, &name, &createTS, &updateTS, &atcEntry,
			&attachment, &uUPIIgnore, &uUPIProductionTree, &uXISComments,
		); err != nil {
			log.Printf("GeTreeProducts: headers scan error: %v", err)
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
	log.Printf("GeTreeProducts: headers scan loop took %s, headers=%d", time.Since(hScanStart), len(headersByCode))

	if err := hRows.Err(); err != nil {
		log.Printf("GeTreeProducts: headers rows.Err(): %v", err)
		return nil, err
	}
	if len(headersByCode) == 0 {
		log.Printf("GeTreeProducts: no headers found, total=%s", time.Since(totalStart))
		return []BomHeaderDTO{}, nil
	}

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

	lQueryStart := time.Now()
	lRows, err := r.Db.Query(linesSQL, args...)
	if err != nil {
		log.Printf("GeTreeProducts: lines Query() error after %s: %v", time.Since(lQueryStart), err)
		return nil, err
	}
	log.Printf("GeTreeProducts: lines Query() took %s", time.Since(lQueryStart))
	defer lRows.Close()

	lScanStart := time.Now()
	lineCount := 0
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
			log.Printf("GeTreeProducts: lines scan error: %v", err)
			return nil, err
		}
		lineCount++

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
	log.Printf("GeTreeProducts: lines scan loop took %s, lines=%d", time.Since(lScanStart), lineCount)

	if err := lRows.Err(); err != nil {
		log.Printf("GeTreeProducts: lines rows.Err(): %v", err)
		return nil, err
	}

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

	log.Printf("GeTreeProducts: DONE total=%s, headers=%d, lines=%d",
		time.Since(totalStart), len(result), lineCount)

	return result, nil
}

func (r *ProductRepository) GetProductStocksData(dto *ProductSkusStockDto) ([]ProductStock, error) {
	totalStart := time.Now()
	log.Printf("GetProductStocksData: start, skus=%d, warehouse=%s", len(dto.Skus), dto.Warehouse)

	if len(dto.Skus) == 0 {
		return nil, fmt.Errorf("sku list cannot be empty")
	}
	if dto.Warehouse == "" {
		return nil, fmt.Errorf("warehouse is required")
	}

	unionParts := make([]string, 0, len(dto.Skus))
	args := make([]any, 0, len(dto.Skus)+1)

	args = append(args, sql.Named("warehouse", dto.Warehouse))

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

	query := fmt.Sprintf(`
WITH ParentSkuList AS (
        %s
),
TreeParents AS (
    SELECT P.sku AS ParentCode
    FROM ParentSkuList AS P
    INNER JOIN OITT AS H WITH (NOLOCK)
        ON H.Code = P.sku
       AND H.TreeType = 'S'
),
ParentChildren AS (
    SELECT H.Code AS ParentCode,
           L.Code AS ChildCode
    FROM OITT AS H WITH (NOLOCK)
    INNER JOIN ITT1 AS L WITH (NOLOCK)
        ON L.Father = H.Code
    WHERE H.TreeType = 'S'
      AND H.Code IN (SELECT sku FROM ParentSkuList)
),
AllItemsForStock AS (
    -- Non-tree parents use their own stock
    SELECT P.sku AS ParentCode,
           P.sku AS ItemCodeToCheck
    FROM ParentSkuList AS P
    WHERE P.sku NOT IN (SELECT ParentCode FROM TreeParents)

    UNION ALL

    -- Tree parents use their children stocks (parent stock = MIN child stock)
    SELECT C.ParentCode,
           C.ChildCode AS ItemCodeToCheck
    FROM ParentChildren AS C
),
StockRaw AS (
    SELECT
        W.ItemCode,
        W.WhsCode,
        W.OnHand,
        W.OnOrder,
        W.IsCommited
    FROM OITW AS W WITH (NOLOCK)
    WHERE W.WhsCode = @warehouse
      AND W.ItemCode IN (SELECT ItemCodeToCheck FROM AllItemsForStock)
),
StockPerParentRows AS (
    SELECT
        A.ParentCode,
        S.ItemCode,
        S.WhsCode,
        S.OnHand,
        S.OnOrder,
        S.IsCommited,
        ROW_NUMBER() OVER (
            PARTITION BY A.ParentCode
            ORDER BY
                CASE WHEN S.OnHand IS NULL THEN 1 ELSE 0 END,
                S.OnHand ASC
        ) AS rn
    FROM AllItemsForStock AS A
    LEFT JOIN StockRaw AS S
      ON S.ItemCode = A.ItemCodeToCheck
)
SELECT
    P.sku AS sku,
    COALESCE(SPR.WhsCode, @warehouse)      AS warehouseCode,
    CAST(SPR.OnHand AS DECIMAL(19,4))      AS stock,
    CAST(SPR.OnOrder AS DECIMAL(19,4))     AS onOrder,
    CAST(SPR.IsCommited AS DECIMAL(19,4))  AS commited
FROM ParentSkuList AS P
LEFT JOIN StockPerParentRows AS SPR
    ON SPR.ParentCode = P.sku
   AND SPR.rn = 1
ORDER BY P.sku;
`, parentSkuUnion)

	qStart := time.Now()
	rows, err := r.Db.Query(query, args...)
	if err != nil {
		log.Printf("GetProductStocksData: Query() error after %s: %v", time.Since(qStart), err)
		return nil, err
	}
	log.Printf("GetProductStocksData: Query() took %s", time.Since(qStart))
	defer rows.Close()

	scanStart := time.Now()
	var result []ProductStock
	for rows.Next() {
		var ps ProductStock
		if err := rows.Scan(
			&ps.SKU,
			&ps.WarehouseCode,
			&ps.Stock,
			&ps.OnOrder,
			&ps.Commited,
		); err != nil {
			log.Printf("GetProductStocksData: scan error: %v", err)
			return nil, err
		}
		result = append(result, ps)
	}
	log.Printf("GetProductStocksData: scan loop took %s, rows=%d", time.Since(scanStart), len(result))

	if err := rows.Err(); err != nil {
		log.Printf("GetProductStocksData: rows.Err(): %v", err)
		return nil, err
	}

	log.Printf("GetProductStocksData: DONE total=%s, rows=%d", time.Since(totalStart), len(result))
	return result, nil
}
