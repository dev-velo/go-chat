package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type parameterGetter interface {
	GetParameter(context.Context, *ssm.GetParameterInput, ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// ResolveAPIKey retrieves the secret once at startup using the runtime's AWS credentials.
// Direct environment configuration needs no AWS access.
func (c *Config) ResolveAPIKey(ctx context.Context) error {
	if !c.SSMFetchKey {
		if c.LLMAPIKey == "" {
			return fmt.Errorf("LLM API key is not configured")
		}
		return nil
	}
	if c.SSMKeyName == "" {
		return fmt.Errorf("SSM_KEY_NAME is required when SSM_FETCH_KEY is true")
	}
	c.LLMAPIKey = ""
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS configuration for LLM API key: %w", err)
	}
	return c.resolveAPIKey(ctx, ssm.NewFromConfig(cfg))
}

func (c *Config) resolveAPIKey(ctx context.Context, client parameterGetter) error {
	result, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(c.SSMKeyName),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("retrieve LLM API key from SSM: %w", err)
	}
	if result == nil || result.Parameter == nil || result.Parameter.Value == nil ||
		strings.TrimSpace(*result.Parameter.Value) == "" {
		return fmt.Errorf("SSM LLM API key parameter has no value")
	}
	c.LLMAPIKey = strings.TrimSpace(*result.Parameter.Value)
	return nil
}
