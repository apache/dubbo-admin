package main

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"swagger/docs"
)

func main() {

	docs.SwaggerInfo.Title = "Dubbo-Admin API"
	docs.SwaggerInfo.Description = "This is a dubbo-admin swagger ui server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "127.0.0.1:38080"
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	r := gin.New()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":38081")
}
