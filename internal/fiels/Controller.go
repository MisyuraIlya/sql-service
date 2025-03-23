package fiels

import (
	"net/http"
	"sql-service/configs"
	"sql-service/pkg/req"
	"sql-service/pkg/res"
)

type FielsControllerDeps struct {
	*configs.Config
	*FileService
}

type FielsController struct {
	*configs.Config
	*FileService
}

func NewFielsController(router *http.ServeMux, deps FielsControllerDeps) *FielsController {
	controller := &FielsController{
		Config:      deps.Config,
		FileService: deps.FileService,
	}
	router.Handle("GET /layer", controller.GetLayer())
	router.Handle("POST /file", controller.FindFile())
	return controller
}

func (Controller *FielsController) GetLayer() http.HandlerFunc {

}

func (Controller *FielsController) FindFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[FindFileDto](&w, r)
		if err != nil {
			return
		}
		data := Controller.FileService.FindFile(body)
		res.Json(w, data, 200)

	}
}
