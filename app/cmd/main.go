package main

import (
	"fmt"
	"log/slog"
	"os"

	"go-chat/app/api"
	"go-chat/app/config"
	"go-chat/app/store"
	"go-chat/app/web"

	"github.com/gin-gonic/gin"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	gin.SetMode(cfg.GinMode)
	exchanges, err := newStore(cfg)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	router := gin.Default()
	api.RegisterRoutes(router, exchanges)
	web.RegisterRoutes(router)

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}

// newStore builds the persistence store based on the config.
func newStore(cfg config.Config) (store.Store, error) {
	switch cfg.StoreBackend {
	case config.StoreMemory:
		return store.NewMemory(), nil
	case config.StoreS3:
		return nil, fmt.Errorf("STORE_BACKEND=%s is not implemented yet", config.StoreS3)
	default:
		return nil, fmt.Errorf("unknown STORE_BACKEND %q (want %s or %s)",
			cfg.StoreBackend, config.StoreMemory, config.StoreS3)
	}
}
