package main

import (
	"fmt"
	"os"

	"go-chat/app/api"
	"go-chat/app/config"
	"go-chat/app/web"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	gin.SetMode(cfg.GinMode)
	router := gin.Default()
	api.RegisterRoutes(router)
	web.RegisterRoutes(router)

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
