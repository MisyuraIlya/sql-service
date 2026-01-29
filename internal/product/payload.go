package product

import "time"

type ProductsDto struct {
	Skus      []string `json:"skus" validate:"required,min=1,dive,required"`
	PriceList *string  `json:"priceList,omitempty"`
	Warehouse string   `json:"warehouse" validate:"required"`
	CardCode  string   `json:"cardCode" validate:"required"`
	Date      string   `json:"date" validate:"required"`
}

type ProductSkusStockDto struct {
	Skus      []string `json:"skus" validate:"required,min=1,dive,required"`
	Warehouse string   `json:"warehouse" validate:"required"`
}

type ProductSkusDto struct {
	Skus []string `json:"skus" validate:"required,min=1,dive,required"`
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
	BPGroupDiscountType  MyNullString  `json:"bpGroupDiscountType"`
	OedgType             MyNullString  `json:"oedgType"`
	OedgValidFor         MyNullBool    `json:"oedgValidFor"`
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

type ProductStock struct {
	SKU           string        `json:"sku"`
	WarehouseCode string        `json:"warehouseCode"`
	Stock         MyNullFloat64 `json:"stock"`
	OnOrder       MyNullFloat64 `json:"onOrder"`
	Commited      MyNullFloat64 `json:"commited"`
}

type BomHeaderDTO struct {
	Code                 string     `json:"Code"`
	TreeType             string     `json:"TreeType"`
	PriceList            *int64     `json:"PriceList,omitempty"`
	Quantity             *float64   `json:"Quantity,omitempty"`
	CreateDate           *time.Time `json:"CreateDate,omitempty"`
	UpdateDate           *time.Time `json:"UpdateDate,omitempty"`
	Transfered           *string    `json:"Transfered,omitempty"`
	DataSource           *string    `json:"DataSource,omitempty"`
	UserSign             *int64     `json:"UserSign,omitempty"`
	SCNCounter           *int64     `json:"SCNCounter,omitempty"`
	DispCurr             *string    `json:"DispCurr,omitempty"`
	ToWH                 *string    `json:"ToWH,omitempty"`
	Object               *string    `json:"Object,omitempty"`
	LogInstac            *int64     `json:"LogInstac,omitempty"`
	UserSign2            *int64     `json:"UserSign2,omitempty"`
	OcrCode              *string    `json:"OcrCode,omitempty"`
	HideComp             *string    `json:"HideComp,omitempty"`
	OcrCode2             *string    `json:"OcrCode2,omitempty"`
	OcrCode3             *string    `json:"OcrCode3,omitempty"`
	OcrCode4             *string    `json:"OcrCode4,omitempty"`
	OcrCode5             *string    `json:"OcrCode5,omitempty"`
	UpdateTime           *int64     `json:"UpdateTime,omitempty"`
	Project              *string    `json:"Project,omitempty"`
	PlAvgSize            *float64   `json:"PlAvgSize,omitempty"`
	Name                 *string    `json:"Name,omitempty"`
	CreateTS             *int64     `json:"CreateTS,omitempty"`
	UpdateTS             *int64     `json:"UpdateTS,omitempty"`
	AtcEntry             *int64     `json:"AtcEntry,omitempty"`
	Attachment           *int64     `json:"Attachment,omitempty"`
	U_UPI_Ignore         *string    `json:"U_UPI_Ignore,omitempty"`
	U_UPI_ProductionTree *string    `json:"U_UPI_ProductionTree,omitempty"`
	U_XIS_Comments       *string    `json:"U_XIS_Comments,omitempty"`

	Lines []BomLineDTO `json:"Lines"`
}

type BomLineDTO struct {
	Father               string   `json:"Father"`
	ChildNum             *int64   `json:"ChildNum,omitempty"`
	VisOrder             *int64   `json:"VisOrder,omitempty"`
	Code                 string   `json:"Code"`
	Quantity             *float64 `json:"Quantity,omitempty"`
	Warehouse            *string  `json:"Warehouse,omitempty"`
	Price                *float64 `json:"Price,omitempty"`
	Currency             *string  `json:"Currency,omitempty"`
	PriceList            *int64   `json:"PriceList,omitempty"`
	OrigPrice            *float64 `json:"OrigPrice,omitempty"`
	OrigCurr             *string  `json:"OrigCurr,omitempty"`
	IssueMthd            *string  `json:"IssueMthd,omitempty"`
	Uom                  *string  `json:"Uom,omitempty"`
	Comment              *string  `json:"Comment,omitempty"`
	LogInstanc           *int64   `json:"LogInstanc,omitempty"`
	Object               *string  `json:"Object,omitempty"`
	OcrCode              *string  `json:"OcrCode,omitempty"`
	OcrCode2             *string  `json:"OcrCode2,omitempty"`
	OcrCode3             *string  `json:"OcrCode3,omitempty"`
	OcrCode4             *string  `json:"OcrCode4,omitempty"`
	OcrCode5             *string  `json:"OcrCode5,omitempty"`
	PrncpInput           *string  `json:"PrncpInput,omitempty"`
	Project              *string  `json:"Project,omitempty"`
	Type                 *string  `json:"Type,omitempty"`
	WipActCode           *string  `json:"WipActCode,omitempty"`
	AddQuantit           *float64 `json:"AddQuantit,omitempty"`
	LineText             *string  `json:"LineText,omitempty"`
	StageId              *int64   `json:"StageId,omitempty"`
	ItemName             *string  `json:"ItemName,omitempty"`
	U_UPI_BaseEl         *string  `json:"U_UPI_BaseEl,omitempty"`
	U_IsVisibleOnWebshop *string  `json:"U_IsVisibleOnWebshop,omitempty"`
	U_InvCalc            *string  `json:"U_InvCalc,omitempty"`
}
