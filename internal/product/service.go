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

// ProductServiceHandler returns the priced rows, or an error the caller MUST
// surface. It deliberately no longer swallows the error into a nil result: a
// failure that looks like an empty success is what allowed a broken price batch
// to reach the catalogue as "price 0" and then transmit to SAP at 0.
func (service *ProductService) ProductServiceHandler(dto *ProductsDto) ([]Product, error) {
	start := time.Now()
	log.Printf("ProductServiceHandler: start, skus=%d, cardCode=%s", len(dto.Skus), dto.CardCode)

	result, err := service.productRepository.GetProducts(dto)
	if err != nil {
		log.Printf("ProductServiceHandler: error after %s: %v", time.Since(start), err)
		return nil, err
	}

	log.Printf("ProductServiceHandler: success, rows=%d, elapsed=%s", len(result), time.Since(start))
	return result, nil
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
