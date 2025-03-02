package product

type ProductsDto struct {
	PriceList string   `json:priceList`
	WareHouse string   `json:wareHouse`
	CardCode  string   `json:cardCode`
	Date      string   `json:date`
	Skus      []string `json:"skus" validate:"required"`
}
