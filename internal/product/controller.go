package product

import (
	"fmt"
	"net/http"
	"sql-service/configs"
	"sql-service/pkg/req"
)

type ProductControllerDeps struct {
	Config         *configs.Config
	ProductService *ProductService
}

type ProductController struct {
	config *configs.Config
}

func NewProductController(router *http.ServeMux, deps ProductControllerDeps) *ProductController {
	controller := &ProductController{
		config: deps.Config,
	}

	router.Handle("POST /products", controller.GetProducts())
	return controller
}

func (Controller *ProductController) GetProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[ProductsDto](&w, r)
		// Get products from database
		if err != nil {
			return
		}

		for _, sku := range body.Skus {
			fmt.Println(sku)
		}

	}
}
