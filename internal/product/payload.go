package product

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
