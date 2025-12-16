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
	router.Handle("POST /openProducts", controller.OpenProducts()) // ✅ POST (body works)

	return controller
}

func (Controller *DocumentController) GetCartesset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[CartessetDto](&w, r)
		if err != nil {
			return
		}

		// Basic validation
		if body.CardCode == "" || body.DateFrom == "" || body.DateTo == "" {
			res.Json(w, map[string]any{
				"error":    "cardCode, dateFrom, dateTo are required",
				"required": []string{"cardCode", "dateFrom", "dateTo"},
			}, http.StatusBadRequest)
			return
		}

		data := Controller.DocumentService.DocumentServiceHandler(body)
		res.Json(w, data, http.StatusOK)
	}
}

func (Controller *DocumentController) OpenProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[AllProductsDto](&w, r)
		if err != nil {
			return
		}

		if body.UserExtId == "" {
			res.Json(w, map[string]any{
				"error":    "userExtId is required",
				"required": []string{"userExtId"},
			}, http.StatusBadRequest)
			return
		}

		data := Controller.DocumentService.OpenProducts(body)
		res.Json(w, data, http.StatusOK)
	}
}
