package product

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"sql-service/pkg/db"
)

// Null Handling for JSON
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

// Product Model (expanded to include new fields)
type Product struct {
	SKU            string        `json:"sku"`
	CardCode       string        `json:"cardCode"`
	PriceList      MyNullFloat64 `json:"priceList"` // ListNum (INT) exposed as float for null-handling; change to a nullable INT if you prefer
	Currency       MyNullString  `json:"currency"`
	PriceListPrice MyNullFloat64 `json:"priceListPrice"`
	OSPPPrice      MyNullFloat64 `json:"osppPrice"`
	OSPPDiscount   MyNullFloat64 `json:"osppDiscount"`
	PromoDiscount  MyNullFloat64 `json:"promoDiscount"`
	WarehouseCode  string        `json:"warehouseCode"`
	Stock          MyNullFloat64 `json:"stock"`
	OnOrder        MyNullFloat64 `json:"onOrder"`
	Commited       MyNullFloat64 `json:"commited"`
	PriceSource    string        `json:"priceSource"`
	FinalPrice     float64       `json:"finalPrice"` // computed, not nullable in SELECT
}

// Repository
type ProductRepository struct{ Db *db.Db }

func NewProductRepository(db *db.Db) *ProductRepository { return &ProductRepository{Db: db} }

// GetProducts implements:
// - Price list resolved from OCRD by CardCode
// - OSPP (per-BP special prices/discounts) with validity window
// - Promotional EDG/EDG1 (Type='A', ObjType='4') discount with validity window
// - Stock (OITW) for requested warehouse
// - Parameterized SKU list via (VALUES ...) to avoid SQL injection
func (r *ProductRepository) GetProducts(dto *ProductsDto) ([]Product, error) {
	if len(dto.Skus) == 0 {
		return nil, fmt.Errorf("sku list cannot be empty")
	}
	// Build (VALUES ...) list parameterized as (@sku0), (@sku1), ...
	vals := make([]string, len(dto.Skus))
	args := []any{
		sql.Named("cardCode", dto.CardCode),
		sql.Named("asOfDate", dto.Date),       // DATE
		sql.Named("warehouse", dto.WareHouse), // NVARCHAR/INT as per schema
	}
	for i, sku := range dto.Skus {
		name := fmt.Sprintf("sku%d", i)
		vals[i] = fmt.Sprintf("(@%s)", name)
		args = append(args, sql.Named(name, sku))
	}
	skuValues := strings.Join(vals, ",")

	// One batch with CTEs; resolve ListNum from OCRD; NOLOCK preserved to match your example
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
    -- OSPP = special per BP
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
PromoDiscount AS (
    -- OEDG/EDG1 Type='A' item-level (ObjType='4')
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
      AND (
            (E.ValidForm IS NULL AND E.ValidTo IS NULL)
         OR (E.ValidForm <= @asOfDate AND (E.ValidTo IS NULL OR E.ValidTo >= @asOfDate))
      )
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
    CAST(PD.PromoDiscount AS DECIMAL(19,4))                          AS PromoDiscount,
    ISNULL(S.warehouseCode, '')                                      AS warehouseCode,
    CAST(S.stock AS DECIMAL(19,4))                                   AS stock,
    CAST(S.onOrder AS DECIMAL(19,4))                                 AS onOrder,
    CAST(S.commited AS DECIMAL(19,4))                                AS commited,
    CASE
        WHEN SP.OSPPPrice IS NOT NULL AND SP.OSPPPrice > 0 THEN N'OSPP explicit price'
        WHEN SP.OSPPDiscount IS NOT NULL THEN N'OSPP discount'
        WHEN PD.PromoDiscount IS NOT NULL THEN N'Promo (EDG Type A)'
        ELSE N'Base price list'
    END                                                              AS PriceSource,
    CAST(
        CASE
            WHEN SP.OSPPPrice IS NOT NULL AND SP.OSPPPrice > 0
                THEN SP.OSPPPrice
            WHEN SP.OSPPDiscount IS NOT NULL
                THEN BP.PriceListPrice * (100.0 - SP.OSPPDiscount) / 100.0
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
LEFT JOIN PromoDiscount AS PD
       ON PD.ItemCode = BP.ItemCode
LEFT JOIN Stock AS S
       ON S.ItemCode = BP.ItemCode
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
