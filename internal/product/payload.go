package product

import (
	"database/sql"
)

type ProductsDto struct {
	Skus      []string `json:"skus" validate:"required,min=1,dive,required"`
	PriceList *string  `json:"priceList,omitempty"`
	Warehouse string   `json:"warehouse" validate:"required"`
	CardCode  string   `json:"cardCode" validate:"required"`
	Date      string   `json:"date" validate:"required"`
}

type ProductsTreeDto struct {
	Skus []string `json:"skus`
}

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
type BomHeader struct {
	Code                 string
	TreeType             string
	PriceList            sql.NullInt64
	Quantity             sql.NullFloat64 // note: OITT column is "Qauntity" (typo in DB); we scan it below
	CreateDate           sql.NullTime
	UpdateDate           sql.NullTime
	Transfered           sql.NullString
	DataSource           sql.NullString
	UserSign             sql.NullInt64
	SCNCounter           sql.NullInt64
	DispCurr             sql.NullString
	ToWH                 sql.NullString
	Object               sql.NullString
	LogInstac            sql.NullInt64
	UserSign2            sql.NullInt64
	OcrCode              sql.NullString
	HideComp             sql.NullString
	OcrCode2             sql.NullString
	OcrCode3             sql.NullString
	OcrCode4             sql.NullString
	OcrCode5             sql.NullString
	UpdateTime           sql.NullTime
	Project              sql.NullString
	PlAvgSize            sql.NullFloat64
	Name                 sql.NullString
	CreateTS             sql.NullTime
	UpdateTS             sql.NullTime
	AtcEntry             sql.NullInt64
	Attachment           sql.NullInt64
	U_UPI_Ignore         sql.NullString
	U_UPI_ProductionTree sql.NullString
	U_XIS_Comments       sql.NullString

	Lines []BomLine
}

type BomLine struct {
	Father               string
	ChildNum             sql.NullInt64
	VisOrder             sql.NullInt64
	Code                 string
	Quantity             sql.NullFloat64
	Warehouse            sql.NullString
	Price                sql.NullFloat64
	Currency             sql.NullString
	PriceList            sql.NullInt64
	OrigPrice            sql.NullFloat64
	OrigCurr             sql.NullString
	IssueMthd            sql.NullString
	Uom                  sql.NullString
	Comment              sql.NullString
	LogInstanc           sql.NullInt64
	Object               sql.NullString
	OcrCode              sql.NullString
	OcrCode2             sql.NullString
	OcrCode3             sql.NullString
	OcrCode4             sql.NullString
	OcrCode5             sql.NullString
	PrncpInput           sql.NullString
	Project              sql.NullString
	Type                 sql.NullString
	WipActCode           sql.NullString
	AddQuantit           sql.NullFloat64
	LineText             sql.NullString
	StageId              sql.NullInt64
	ItemName             sql.NullString
	U_UPI_BaseEl         sql.NullString
	U_IsVisibleOnWebshop sql.NullString
	U_InvCalc            sql.NullString
}
