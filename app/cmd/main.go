package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go-chat/app/api"
	"go-chat/app/config"
	"go-chat/app/llm"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = cfg.ResolveAPIKey(ctx)
	cancel()
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

	model, err := newLLM(cfg)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	router := gin.New()
	router.Use(api.Logging(), api.Recovery())
	api.RegisterRoutes(router, exchanges, model)
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

// newLLM builds the provider named by the config.
func newLLM(cfg config.Config) (llm.Client, error) {
	switch cfg.LLMProvider {
	case config.ProviderFake:
		return llm.NewFake(cfg.LLMAPIKey, cfg.LLMModel)
	case config.ProviderAnthropic:
		return nil, fmt.Errorf("LLM_PROVIDER=%s is not implemented yet", config.ProviderAnthropic)
	case config.ProviderOpenAI:
		return llm.NewOpenAI(cfg.LLMAPIKey, cfg.LLMModel)
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (want %s, %s, or %s)",
			cfg.LLMProvider, config.ProviderFake, config.ProviderAnthropic, config.ProviderOpenAI)
	}
}
