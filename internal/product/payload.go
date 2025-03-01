package product

type ProductsDto struct {
	Skus []string `json:"skus" validate:"required"`
}
