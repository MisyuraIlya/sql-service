package fiels

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sql-service/configs"
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

	router.Handle("GET /images", controller.GetAllImages())
	router.Handle("GET /image/", controller.GetImage())

	router.Handle("GET /productlinearts", controller.GetProductLineArtsImages())
	router.Handle("GET /productlineart/", controller.GetProductLineArt())

	return controller
}

func (controller *FielsController) GetAllImages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		folderPath := controller.Config.ImagesPath
		images, err := controller.FileService.ListImages(folderPath)
		if err != nil {
			res.Json(w, "Unable to fetch images", http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]interface{}{"images": images}, http.StatusOK)
	}
}

func (controller *FielsController) GetImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := "/image/"
		fileName := strings.TrimPrefix(r.URL.Path, prefix)
		if fileName == "" {
			res.Json(w, "File name is required", http.StatusBadRequest)
			return
		}

		folderPath := controller.Config.ImagesPath
		filePath := filepath.Join(folderPath, fileName)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			res.Json(w, "Image not found", http.StatusNotFound)
			return
		}

		ext := strings.ToLower(filepath.Ext(fileName))
		switch ext {
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		case ".bmp":
			w.Header().Set("Content-Type", "image/bmp")
		default:
			res.Json(w, "Unsupported image format", http.StatusUnsupportedMediaType)
			return
		}

		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
		http.ServeFile(w, r, filePath)
	}
}

func (controller *FielsController) GetProductLineArtsImages() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		folderPath := controller.Config.ProductLineArtsPath
		images, err := controller.FileService.ListImages(folderPath)
		if err != nil {
			res.Json(w, "Unable to fetch product line arts images", http.StatusInternalServerError)
			return
		}
		res.Json(w, map[string]interface{}{"images": images}, http.StatusOK)
	}
}

func (controller *FielsController) GetProductLineArt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := "/productlineart/"
		fileName := strings.TrimPrefix(r.URL.Path, prefix)
		if fileName == "" {
			res.Json(w, "File name is required", http.StatusBadRequest)
			return
		}

		folderPath := controller.Config.ProductLineArtsPath
		filePath := filepath.Join(folderPath, fileName)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			res.Json(w, "Image not found", http.StatusNotFound)
			return
		}

		ext := strings.ToLower(filepath.Ext(fileName))
		switch ext {
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		case ".bmp":
			w.Header().Set("Content-Type", "image/bmp")
		default:
			res.Json(w, "Unsupported image format", http.StatusUnsupportedMediaType)
			return
		}

		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
		http.ServeFile(w, r, filePath)
	}
}
