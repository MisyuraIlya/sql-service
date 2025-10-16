package product

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"sql-service/pkg/db"
)

// ---- Null handling for JSON ----

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

// ---- Model ----

type Product struct {
	SKU                  string        `json:"sku"`
	CardCode             string        `json:"cardCode"`
	PriceList            MyNullFloat64 `json:"priceList"`
	Currency             MyNullString  `json:"currency"`
	PriceListPrice       MyNullFloat64 `json:"priceListPrice"`
	OSPPPrice            MyNullFloat64 `json:"osppPrice"`
	OSPPDiscount         MyNullFloat64 `json:"osppDiscount"`
	BPGroupDiscount      MyNullFloat64 `json:"bpGroupDiscount"`
	ManufacturerName     MyNullString  `json:"manufacturerName"`
	ManufacturerDiscount MyNullFloat64 `json:"manufacturerDiscount"`
	PromoDiscount        MyNullFloat64 `json:"promoDiscount"`
	WarehouseCode        string        `json:"warehouseCode"`
	Stock                MyNullFloat64 `json:"stock"`
	OnOrder              MyNullFloat64 `json:"onOrder"`
	Commited             MyNullFloat64 `json:"commited"`
	PriceSource          string        `json:"priceSource"`
	FinalPrice           float64       `json:"finalPrice"`
}

// ---- Repository ----

type ProductRepository struct{ Db *db.Db }

func NewProductRepository(db *db.Db) *ProductRepository { return &ProductRepository{Db: db} }

func (r *ProductRepository) GetProducts(dto *ProductsDto) ([]Product, error) {
	if len(dto.Skus) == 0 {
		return nil, fmt.Errorf("sku list cannot be empty")
	}

	vals := make([]string, len(dto.Skus))
	args := []any{
		sql.Named("cardCode", dto.CardCode),
		sql.Named("userExtId", dto.CardCode),
		sql.Named("asOfDate", dto.Date),
		sql.Named("warehouse", dto.WareHouse),
	}
	for i, sku := range dto.Skus {
		name := fmt.Sprintf("sku%d", i)
		vals[i] = fmt.Sprintf("(@%s)", name)
		args = append(args, sql.Named(name, sku))
	}
	skuValues := strings.Join(vals, ",")

	query := fmt.Sprintf(`
WITH SkuList AS (
    SELECT v.sku FROM (VALUES %s) AS v(sku)
),
Cust AS (
    SELECT TOP 1 T5.ListNum
    FROM OCRD AS T5 WITH (NOLOCK)
    WHERE T5.CardCode = @cardCode
),
BasePrice AS (
    SELECT
        OITM.ItemCode,
        OITM.FirmCode,                    -- Manufacturer code (OMRC.FirmCode)
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
    -- OSPP = special per BP (has validity window)
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
BPGroupDiscount AS (
    -- BP-specific Discount Group rules (Type='S') - Manufacturer level (ObjType='43')
    -- NOTE: no ValidFrom/ValidTo in OEDG; only ValidFor flag.
    SELECT
        BP.ItemCode,
        MAX(E1.Discount) AS BPGroupDiscount
    FROM BasePrice AS BP
    INNER JOIN OEDG AS E WITH (NOLOCK)
        ON E.Type = 'S'
       AND E.ObjCode = @cardCode   -- rules defined for this BP
       AND E.ValidFor = 'Y'
    INNER JOIN EDG1 AS E1 WITH (NOLOCK)
        ON E1.AbsEntry = E.AbsEntry
       AND E1.ObjType = '43'       -- Manufacturer
       AND TRY_CAST(E1.ObjKey AS INT) = BP.FirmCode
    GROUP BY BP.ItemCode
),
PromoDiscount AS (
    -- Global promos: OEDG Type='A' with item lines (ObjType='4')
    -- NOTE: no ValidFrom/ValidTo filters here either.
    SELECT
        I.ItemCode,
        E1.Discount AS PromoDiscount
    FROM OEDG AS E WITH (NOLOCK)
    INNER JOIN EDG1 AS E1 WITH (NOLOCK)
        ON E1.AbsEntry = E.AbsEntry
       AND E1.ObjType = '4'        -- Item
    INNER JOIN OITM AS I WITH (NOLOCK)
        ON I.ItemCode = E1.ObjKey
    WHERE E.ValidFor = 'Y'
      AND E.Type = 'A'
),
ManufacturerDiscount AS (
    -- Per-item manufacturer discount for this userExtId:
    -- 1) Take item -> FirmCode from OITM (via BasePrice)
    -- 2) Find OEDG Type='S' rules for @userExtId with EDG1 ObjType='43' where ObjKey = FirmCode
    -- 3) Return ManufacturerName + Discount per ItemCode
    SELECT
        BP.ItemCode,
        M.FirmName             AS ManufacturerName,
        E1.Discount            AS DiscountPercentage
    FROM BasePrice AS BP
    INNER JOIN OEDG AS E WITH (NOLOCK)
        ON E.Type = 'S'
       AND E.ObjCode = @userExtId   -- use external user id from params
       AND E.ValidFor = 'Y'
    INNER JOIN EDG1 AS E1 WITH (NOLOCK)
        ON E1.AbsEntry = E.AbsEntry
       AND E1.ObjType = '43'        -- Manufacturer
       AND TRY_CAST(E1.ObjKey AS INT) = BP.FirmCode
    LEFT JOIN OMRC AS M WITH (NOLOCK)
        ON M.FirmCode = BP.FirmCode
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
    CAST(BPG.BPGroupDiscount AS DECIMAL(19,4))                       AS BPGroupDiscount,
    MD.ManufacturerName                                              AS ManufacturerName,
    CAST(MD.DiscountPercentage AS DECIMAL(19,4))                     AS ManufacturerDiscount,
    CAST(PD.PromoDiscount AS DECIMAL(19,4))                          AS PromoDiscount,
    ISNULL(S.warehouseCode, '')                                      AS warehouseCode,
    CAST(S.stock AS DECIMAL(19,4))                                   AS stock,
    CAST(S.onOrder AS DECIMAL(19,4))                                 AS onOrder,
    CAST(S.commited AS DECIMAL(19,4))                                AS commited,
    CASE
        WHEN SP.OSPPPrice IS NOT NULL AND SP.OSPPPrice > 0 THEN N'OSPP explicit price'
        WHEN SP.OSPPDiscount IS NOT NULL THEN N'OSPP discount'
        WHEN BPG.BPGroupDiscount IS NOT NULL THEN N'BP discount group (manufacturer)'
        WHEN PD.PromoDiscount IS NOT NULL THEN N'Promo (EDG Type A)'
        ELSE N'Base price list'
    END                                                              AS PriceSource,
    CAST(
        CASE
            WHEN SP.OSPPPrice IS NOT NULL AND SP.OSPPPrice > 0
                THEN SP.OSPPPrice
            WHEN SP.OSPPDiscount IS NOT NULL
                THEN BP.PriceListPrice * (100.0 - SP.OSPPDiscount) / 100.0
            WHEN BPG.BPGroupDiscount IS NOT NULL
                THEN BP.PriceListPrice * (100.0 - BPG.BPGroupDiscount) / 100.0
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
LEFT JOIN BPGroupDiscount AS BPG
       ON BPG.ItemCode = BP.ItemCode
LEFT JOIN PromoDiscount AS PD
       ON PD.ItemCode = BP.ItemCode
LEFT JOIN Stock AS S
       ON S.ItemCode = BP.ItemCode
LEFT JOIN ManufacturerDiscount AS MD
       ON MD.ItemCode = BP.ItemCode
ORDER BY BP.ItemCode;
`, skuValues)

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
