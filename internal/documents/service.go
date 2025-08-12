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
	}
	return result
}
