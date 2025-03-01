package product

type ProductService struct {
	productRepository *ProductRepository
}

func NewProductService(repo *ProductRepository) *ProductService {
	return &ProductService{
		productRepository: repo,
	}
}

func PriceHandler() {

}

func StockHandler() {

}
