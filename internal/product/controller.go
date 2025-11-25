package product

import (
	"net/http"
	"sql-service/configs"
	"sql-service/pkg/req"
	"sql-service/pkg/res"
)

type ProductControllerDeps struct {
	*configs.Config
	*ProductService
}

type ProductController struct {
	*configs.Config
	*ProductService
}

func NewProductController(router *http.ServeMux, deps ProductControllerDeps) *ProductController {
	controller := &ProductController{
		Config:         deps.Config,
		ProductService: deps.ProductService,
	}

	router.Handle("POST /products", controller.GetProducts())
	router.Handle("POST /productTree", controller.GetProductTree())
	router.Handle("POST /productStock", controller.GetProductStcok())
	return controller
}

func (Controller *ProductController) GetProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[ProductsDto](&w, r)
		// Get products from database
		if err != nil {
			return
		}
		data := Controller.ProductService.ProductServiceHandler(body)
		res.Json(w, data, 200)

	}
}

func (Controller *ProductController) GetProductTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[ProductSkusDto](&w, r)
		if err != nil {
			return
		}
		data := Controller.ProductService.ProductTreeHandler(body)
		res.Json(w, data, 200)

	}
}

func (Controller *ProductController) GetProductStcok() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[ProductSkusStockDto](&w, r)
		if err != nil {
			return
		}
		data := Controller.ProductService.ProductStocks(body)
		res.Json(w, data, 200)
	}
}
