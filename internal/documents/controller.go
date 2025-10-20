package documents

import (
	"net/http"
	"sql-service/configs"
	"sql-service/pkg/req"
	"sql-service/pkg/res"
)

type DocumentControllerDeps struct {
	*configs.Config
	*DocumentService
}

type DocumentController struct {
	*configs.Config
	*DocumentService
}

func NewDocumentController(router *http.ServeMux, deps DocumentControllerDeps) *DocumentController {
	controller := &DocumentController{
		Config:          deps.Config,
		DocumentService: deps.DocumentService,
	}

	router.Handle("POST /cartesset", controller.GetCartesset())
	router.Handle("GET /openProducts", controller.OpenProducts())
	return controller
}

func (Controller *DocumentController) GetCartesset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[CartessetDto](&w, r)
		// Get products from database
		if err != nil {
			return
		}
		data := Controller.DocumentService.DocumentServiceHandler(body)
		res.Json(w, data, 200)

	}
}

func (Controller *DocumentController) OpenProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := Controller.DocumentService.OpenProducts()
		res.Json(w, data, 200)

	}
}
