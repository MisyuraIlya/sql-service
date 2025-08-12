package main

import (
	"fmt"
	"log"
	"net/http"
	"sql-service/configs"
	"sql-service/internal/documents"
	"sql-service/internal/fiels"
	"sql-service/internal/product"
	"sql-service/pkg/db"
)

func App() http.Handler {
	conf := configs.LoadConfig()

	conn, err := db.NewConnection(conf)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	router := http.NewServeMux()

	// repositories
	productRepository := product.NewProductRepository(conn)
	documentsRepository := documents.NewDocumentRepository(conn)

	// services
	productService := product.NewProductService(productRepository)
	documentService := documents.NewDocumentService(documentsRepository)
	filesService := fiels.NewFilesService()

	// controllers
	product.NewProductController(router, product.ProductControllerDeps{
		Config:         conf,
		ProductService: productService,
	})

	documents.NewDocumentController(router, documents.DocumentControllerDeps{
		Config:          conf,
		DocumentService: documentService,
	})

	fiels.NewFielsController(router, fiels.FielsControllerDeps{
		Config:      conf,
		FileService: filesService,
	})

	return router
}

func main() {
	app := App()
	server := http.Server{
		Addr:    ":2222",
		Handler: app,
	}
	fmt.Println("Server is listening on port 2222")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
