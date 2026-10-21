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

	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
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
	err = cfg.ResolveSecrets(ctx)
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

	router := newRouter(exchanges, model, cfg.AppUserName, cfg.AppUserPass)

	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(ginadapter.NewV2(router).ProxyWithContext)
		return
	}

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}

}

func newRouter(exchanges store.Store, model llm.Client, username, password string) *gin.Engine {
	router := gin.New()
	router.Use(api.Logging(), api.Recovery(), api.BasicAuth(username, password))
	api.RegisterRoutes(router, exchanges, model)
	web.RegisterRoutes(router)
	return router
}

// newStore builds the persistence store based on the config.
func newStore(cfg config.Config) (store.Store, error) {
	switch cfg.StoreBackend {
	case config.StoreMemory:
		return store.NewMemory(), nil
	case config.StoreS3:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return store.NewS3(ctx, cfg.S3BucketName, cfg.S3BucketPath)
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
		return llm.NewAnthropic(cfg.LLMAPIKey, cfg.LLMModel)
	case config.ProviderOpenAI:
		return llm.NewOpenAI(cfg.LLMAPIKey, cfg.LLMModel)
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (want %s, %s, or %s)",
			cfg.LLMProvider, config.ProviderFake, config.ProviderAnthropic, config.ProviderOpenAI)
	}
}
