package main

import (
	"fmt"
	"net/http"
	"sql-service/configs"
	"sql-service/internal/product"
	"sql-service/pkg/db"
	"sql-service/pkg/redis"
)

func App() http.Handler {
	conf := configs.LoadConfig()
	redis := redis.NewRedis(conf)
	db := db.NewDb(conf)
	router := http.NewServeMux()

	//repositories
	productRepository := product.NewProductRepository(db, redis)

	//services
	productService := product.NewProductService(productRepository)

	//controllers
	product.NewProductController(router, product.ProductControllerDeps{
		Config:         conf,
		ProductService: productService,
	})
	return router
}

func main() {
	app := App()
	server := http.Server{
		Addr:    ":8080",
		Handler: app,
	}
	fmt.Println("Server is listening on port 8080")
	server.ListenAndServe()
}
