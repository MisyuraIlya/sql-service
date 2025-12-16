package documents

import "fmt"

type DocumentService struct {
	documentRrepository *DocumentRrepository
}

func NewDocumentService(repo *DocumentRrepository) *DocumentService {
	return &DocumentService{
		documentRrepository: repo,
	}
}

func (service *DocumentService) DocumentServiceHandler(dto *CartessetDto) []Cartesset {
	result, err := service.documentRrepository.GetCartesset(dto)
	if err != nil {
		fmt.Println("error", err.Error())
		return []Cartesset{} // ✅ never nil
	}
	return result
}

func (service *DocumentService) OpenProducts(dto *AllProductsDto) []OpenProducts {
	result, err := service.documentRrepository.GetOpenProducts(dto)
	if err != nil {
		fmt.Println("error", err.Error())
		return []OpenProducts{} // ✅ never nil
	}
	if result == nil {
		return []OpenProducts{} // ✅ extra safety
	}
	return result
}
