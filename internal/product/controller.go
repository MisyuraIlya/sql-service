package product

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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
		reqStart := time.Now()
		log.Printf("[/products] start")

		body, err := req.HandleBody[ProductsDto](&w, r)
		if err != nil {
			log.Printf("[/products] failed to parse body: %v (elapsed=%s)", err, time.Since(reqStart))
			return
		}
		log.Printf("[/products] body parsed (elapsed=%s), skus=%d", time.Since(reqStart), len(body.Skus))

		if payload, err := json.Marshal(body); err == nil {
			log.Printf("[/products] request body dump: %s", string(payload))
		} else {
			log.Printf("[/products] failed to marshal body for logging: %v", err)
		}

		data := Controller.ProductService.ProductServiceHandler(body)
		log.Printf("[/products] service done (elapsed=%s), rows=%d", time.Since(reqStart), len(data))

		res.Json(w, data, http.StatusOK)
		log.Printf("[/products] response sent (total=%s)", time.Since(reqStart))
	}
}

func (Controller *ProductController) GetProductTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqStart := time.Now()
		log.Printf("[/productTree] start")

		body, err := req.HandleBody[ProductSkusDto](&w, r)
		if err != nil {
			log.Printf("[/productTree] failed to parse body: %v (elapsed=%s)", err, time.Since(reqStart))
			return
		}
		log.Printf("[/productTree] body parsed (elapsed=%s), skus=%d", time.Since(reqStart), len(body.Skus))

		data := Controller.ProductService.ProductTreeHandler(body)
		log.Printf("[/productTree] service done (elapsed=%s), headers=%d", time.Since(reqStart), len(data))

		res.Json(w, data, http.StatusOK)
		log.Printf("[/productTree] response sent (total=%s)", time.Since(reqStart))
	}
}

func (Controller *ProductController) GetProductStcok() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqStart := time.Now()
		log.Printf("[/productStock] start")

		body, err := req.HandleBody[ProductSkusStockDto](&w, r)
		if err != nil {
			log.Printf("[/productStock] failed to parse body: %v (elapsed=%s)", err, time.Since(reqStart))
			return
		}
		log.Printf("[/productStock] body parsed (elapsed=%s), skus=%d", time.Since(reqStart), len(body.Skus))

		data := Controller.ProductService.ProductStocks(body)
		log.Printf("[/productStock] service done (elapsed=%s), rows=%d", time.Since(reqStart), len(data))

		res.Json(w, data, http.StatusOK)
		log.Printf("[/productStock] response sent (total=%s)", time.Since(reqStart))
	}
}
