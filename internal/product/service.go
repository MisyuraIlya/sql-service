package product

import "fmt"

type ProductService struct {
	productRepository *ProductRepository
}

func NewProductService(repo *ProductRepository) *ProductService {
	return &ProductService{
		productRepository: repo,
	}
}

func (service *ProductService) ProductServiceHandler(dto *ProductsDto) []Product {
	result, err := service.productRepository.GetProducts(dto)
	if err != nil {
		fmt.Println("error", err.Error())
	}
	return result
}

func (service *ProductService) ProductTreeHandler(dto *ProductsTreeDto) []BomHeaderDTO {
	result, err := service.productRepository.GeTreeProducts(dto)
	if err != nil {
		fmt.Println("error", err.Error())
	}
	return result
}
