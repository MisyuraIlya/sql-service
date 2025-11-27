package product

import (
	"log"
	"time"
)

type ProductService struct {
	productRepository *ProductRepository
}

func NewProductService(repo *ProductRepository) *ProductService {
	return &ProductService{
		productRepository: repo,
	}
}

func (service *ProductService) ProductServiceHandler(dto *ProductsDto) []Product {
	start := time.Now()
	log.Printf("ProductServiceHandler: start, skus=%d, cardCode=%s", len(dto.Skus), dto.CardCode)

	result, err := service.productRepository.GetProducts(dto)
	if err != nil {
		log.Printf("ProductServiceHandler: error after %s: %v", time.Since(start), err)
		return nil
	}

	log.Printf("ProductServiceHandler: success, rows=%d, elapsed=%s", len(result), time.Since(start))
	return result
}

func (service *ProductService) ProductTreeHandler(dto *ProductSkusDto) []BomHeaderDTO {
	start := time.Now()
	log.Printf("ProductTreeHandler: start, skus=%d", len(dto.Skus))

	result, err := service.productRepository.GeTreeProducts(dto)
	if err != nil {
		log.Printf("ProductTreeHandler: error after %s: %v", time.Since(start), err)
		return nil
	}

	log.Printf("ProductTreeHandler: success, headers=%d, elapsed=%s", len(result), time.Since(start))
	return result
}

func (service *ProductService) ProductStocks(dto *ProductSkusStockDto) []ProductStock {
	start := time.Now()
	log.Printf("ProductStocks: start, skus=%d, warehouse=%s", len(dto.Skus), dto.Warehouse)

	result, err := service.productRepository.GetProductStocksData(dto)
	if err != nil {
		log.Printf("ProductStocks: error after %s: %v", time.Since(start), err)
		return nil
	}

	log.Printf("ProductStocks: success, rows=%d, elapsed=%s", len(result), time.Since(start))
	return result
}
