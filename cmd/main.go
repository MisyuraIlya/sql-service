package main

import (
	"fmt"
	"log"
	"net/http"
	"sql-service/configs"
	"sql-service/internal/product"
	"sql-service/pkg/db"
)

func App() http.Handler {
	conf := configs.LoadConfig()

	// Capture both the connection and the error
	conn, err := db.NewConnection(conf)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	router := http.NewServeMux()

	// repositories
	productRepository := product.NewProductRepository(conn)

	// services
	productService := product.NewProductService(productRepository)

	// controllers
	product.NewProductController(router, product.ProductControllerDeps{
		Config:         conf,
		ProductService: productService,
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
