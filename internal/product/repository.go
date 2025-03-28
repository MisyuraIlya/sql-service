package product

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"sql-service/pkg/db"
)

// Null Handling for JSON
type MyNullFloat64 struct {
	sql.NullFloat64
}

func (n MyNullFloat64) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.Float64)
	}
	return []byte("null"), nil
}

type MyNullString struct {
	sql.NullString
}

func (s MyNullString) MarshalJSON() ([]byte, error) {
	if s.Valid {
		return json.Marshal(s.String)
	}
	return []byte("null"), nil
}

// Product Model
type Product struct {
	PriceListPrice      MyNullFloat64 `json:"priceListPrice"`
	Currency            MyNullString  `json:"currency"`
	SpecialPriceLvl1    MyNullFloat64 `json:"specialPriceLvl1"`
	SpecialPriceLvl2    MyNullFloat64 `json:"specialPriceLvl2"`
	SpecialDiscountLvl1 MyNullFloat64 `json:"specialDiscountLvl1"`
	SpecialDiscountLvl2 MyNullFloat64 `json:"specialDiscountLvl2"`
	SKU                 string        `json:"sku"`
	WarehouseCode       string        `json:"warehouseCode"`
	Stock               MyNullFloat64 `json:"stock"`
	OnOrder             MyNullFloat64 `json:"onOrder"`
	Commited            MyNullFloat64 `json:"commited"`
}

// Repository
type ProductRepository struct {
	Db *db.Db
}

func NewProductRepository(db *db.Db) *ProductRepository {
	return &ProductRepository{
		Db: db,
	}
}

// Fetch Products
func (r *ProductRepository) GetProducts(dto *ProductsDto) ([]Product, error) {
	if len(dto.Skus) == 0 {
		return nil, fmt.Errorf("sku list cannot be empty")
	}

	// Step 1: Construct Temporary Table Query
	insertSkuQuery := "DECLARE @SkuList TABLE (sku NVARCHAR(50));"
	for _, sku := range dto.Skus {
		insertSkuQuery += fmt.Sprintf("INSERT INTO @SkuList (sku) VALUES ('%s');", sku)
	}

	// Step 2: Construct Main Query
	query := insertSkuQuery + `
	SELECT 
		ITM1.[Price] AS [PriceListPrice],
		ITM1.[Currency] AS [Currency],
		OSPP.[Price] AS [SpecialPriceLvl1],
		SPP1.[Price] AS [SpecialPriceLvl2],
		OSPP.[Discount] AS [SpecialDiscountLvl1],
		SPP1.[Discount] AS [SpecialDiscountLvl2],
		OITW.[ItemCode] AS [sku],
		OITW.[WhsCode] AS [warehouseCode],
		OITW.[OnHand] AS [stock],
		OITW.[OnOrder] AS [onOrder],
		OITW.[IsCommited] AS [commited]
	FROM [OITM] OITM
	LEFT JOIN [ITM1] ITM1 
		ON OITM.[ItemCode] = ITM1.[ItemCode]
	LEFT JOIN [OSPP] OSPP 
		ON OITM.[ItemCode] = OSPP.[ItemCode]
		AND ITM1.[PriceList] = OSPP.[ListNum]
		AND (
			(OSPP.[ValidFrom] IS NULL AND OSPP.[ValidTo] IS NULL)
			OR (OSPP.[ValidFrom] <= @date1 AND OSPP.[ValidTo] IS NULL)
			OR (OSPP.[ValidFrom] <= @date2 AND OSPP.[ValidTo] >= @date3)
		)
	LEFT JOIN [SPP1] SPP1 
		ON OITM.[ItemCode] = SPP1.[ItemCode]
		AND OSPP.[ListNum] = SPP1.[ListNum]
		AND (
			(SPP1.[FromDate] IS NULL AND SPP1.[ToDate] IS NULL)
			OR (SPP1.[FromDate] <= @date4 AND SPP1.[ToDate] IS NULL)
			OR (SPP1.[FromDate] <= @date5 AND SPP1.[ToDate] >= @date6)
		)
	LEFT JOIN OITW
		ON OITM.[ItemCode] = OITW.[ItemCode]
		AND OITW.[WhsCode] = @warehouse
	WHERE 
		OITM.[ItemCode] IN (SELECT sku FROM @SkuList)
		AND ITM1.[PriceList] = @priceList
		AND (
			OSPP.[ListNum] IS NULL 
			OR (
				(OSPP.[CardCode] IS NULL OR OSPP.[CardCode] = @cardCode)
				AND OSPP.[ListNum] = @priceList
			)
		);
	`

	// Step 3: Define Parameters
	args := []interface{}{
		sql.Named("date1", dto.Date),
		sql.Named("date2", dto.Date),
		sql.Named("date3", dto.Date),
		sql.Named("date4", dto.Date),
		sql.Named("date5", dto.Date),
		sql.Named("date6", dto.Date),
		sql.Named("warehouse", dto.WareHouse),
		sql.Named("priceList", dto.PriceList),
		sql.Named("cardCode", dto.CardCode),
	}

	// Step 4: Execute Query
	rows, err := r.Db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Step 5: Process Result
	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.PriceListPrice,
			&p.Currency,
			&p.SpecialPriceLvl1,
			&p.SpecialPriceLvl2,
			&p.SpecialDiscountLvl1,
			&p.SpecialDiscountLvl2,
			&p.SKU,
			&p.WarehouseCode,
			&p.Stock,
			&p.OnOrder,
			&p.Commited,
		); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
