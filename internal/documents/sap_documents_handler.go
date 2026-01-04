package documents

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sql-service/pkg/res"
)

type requestError struct {
	status  int
	message string
	details string
}

func (e requestError) Error() string { return e.message }

func (Controller *DocumentController) GetSapDocuments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := parseSapDocumentsQuery(r)
		if err != nil {
			status := http.StatusBadRequest
			message := "invalid request"
			details := err.Error()
			var reqErr requestError
			if errors.As(err, &reqErr) {
				status = reqErr.status
				message = reqErr.message
				details = reqErr.details
			}
			res.Json(w, map[string]any{"error": message, "details": details}, status)
			return
		}

		response, err := Controller.DocumentService.GetSapDocuments(r.Context(), *query)
		if err != nil {
			res.Json(w, map[string]any{"error": "failed to fetch documents", "details": err.Error()}, http.StatusInternalServerError)
			return
		}

		if response.Items == nil {
			response.Items = []map[string]any{}
		}

		res.Json(w, response, http.StatusOK)
	}
}

func parseSapDocumentsQuery(r *http.Request) (*SapDocumentsQuery, error) {
	values := r.URL.Query()

	dateFromStr := strings.TrimSpace(values.Get("dateFrom"))
	dateToStr := strings.TrimSpace(values.Get("dateTo"))
	if dateFromStr == "" || dateToStr == "" {
		return nil, requestError{
			status:  http.StatusBadRequest,
			message: "dateFrom and dateTo are required",
			details: "dateFrom and dateTo are required",
		}
	}

	dateFrom, err := time.Parse("2006-01-02", dateFromStr)
	if err != nil {
		return nil, requestError{
			status:  http.StatusBadRequest,
			message: "invalid dateFrom",
			details: err.Error(),
		}
	}

	dateTo, err := time.Parse("2006-01-02", dateToStr)
	if err != nil {
		return nil, requestError{
			status:  http.StatusBadRequest,
			message: "invalid dateTo",
			details: err.Error(),
		}
	}

	if dateFrom.After(dateTo) {
		return nil, requestError{
			status:  http.StatusBadRequest,
			message: "dateFrom must be before or equal to dateTo",
			details: "dateFrom must be before or equal to dateTo",
		}
	}

	var cardCode *string
	if value := strings.TrimSpace(values.Get("cardCode")); value != "" {
		cardCode = &value
	}

	var warehouseCode *string
	if value := strings.TrimSpace(values.Get("warehouseCode")); value != "" {
		warehouseCode = &value
	}

	var docStatus *string
	if value := strings.TrimSpace(values.Get("DocStatus")); value != "" {
		if value != "O" && value != "C" {
			return nil, requestError{
				status:  http.StatusBadRequest,
				message: "DocStatus must be O or C",
				details: "DocStatus must be O or C",
			}
		}
		docStatus = &value
	}

	sortBy := strings.TrimSpace(values.Get("sortBy"))
	if sortBy == "" {
		sortBy = "DocDate"
	}
	switch strings.ToLower(sortBy) {
	case "docdate":
		sortBy = "DocDate"
	case "docentry":
		sortBy = "DocEntry"
	default:
		return nil, requestError{
			status:  http.StatusBadRequest,
			message: "invalid sortBy",
			details: "sortBy must be DocDate or DocEntry",
		}
	}

	sortDir := strings.TrimSpace(values.Get("sortDir"))
	if sortDir == "" {
		sortDir = "desc"
	}
	switch strings.ToLower(sortDir) {
	case "asc":
		sortDir = "asc"
	case "desc":
		sortDir = "desc"
	default:
		return nil, requestError{
			status:  http.StatusBadRequest,
			message: "invalid sortDir",
			details: "sortDir must be asc or desc",
		}
	}

	page := 1
	if value := strings.TrimSpace(values.Get("page")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return nil, requestError{
				status:  http.StatusBadRequest,
				message: "invalid page",
				details: "page must be an integer >= 1",
			}
		}
		page = parsed
	}

	pageSize := 50
	if value := strings.TrimSpace(values.Get("pageSize")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			return nil, requestError{
				status:  http.StatusBadRequest,
				message: "invalid pageSize",
				details: "pageSize must be an integer between 1 and 200",
			}
		}
		pageSize = parsed
	}

	return &SapDocumentsQuery{
		CardCode:      cardCode,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		WarehouseCode: warehouseCode,
		DocStatus:     docStatus,
		SortBy:        sortBy,
		SortDir:       sortDir,
		Page:          page,
		PageSize:      pageSize,
	}, nil
}
