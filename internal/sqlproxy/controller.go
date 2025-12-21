package sqlproxy

import (
	"net/http"

	"sql-service/pkg/req"
	"sql-service/pkg/res"
)

type ControllerDeps struct {
	*Service
}

type Controller struct {
	*Service
}

func NewController(router *http.ServeMux, deps ControllerDeps) *Controller {
	c := &Controller{Service: deps.Service}
	router.Handle("POST /sql", c.Run())
	return c
}

func (c *Controller) Run() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := req.HandleBody[QueryRequest](&w, r)
		if err != nil {
			return
		}

		if body.DBName == "" {
			res.Json(w, map[string]any{"error": "dbName is required"}, http.StatusBadRequest)
			return
		}
		if body.DB.Server == "" || body.DB.Database == "" || body.DB.User == "" {
			res.Json(w, map[string]any{
				"error": "db.server, db.database, db.user are required",
			}, http.StatusBadRequest)
			return
		}

		out, err := c.Service.Run(r.Context(), body)
		if err != nil {
			res.Json(w, map[string]any{"error": err.Error()}, http.StatusBadRequest)
			return
		}

		res.Json(w, out, http.StatusOK)
	}
}
