package main

import (
	"go-chat/app/api"
	"go-chat/app/config"
	"go-chat/app/web"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	router := gin.Default()
	api.RegisterRoutes(router)
	web.RegisterRoutes(router)

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
