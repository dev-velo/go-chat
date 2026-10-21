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

// ResolveSecrets loads all remote credentials once at startup. Memory mode
// needs no AWS access; S3 mode never falls back to environment secrets.
func (c *Config) ResolveSecrets(ctx context.Context) error {
	switch c.StoreBackend {
	case StoreMemory:
		if c.LLMAPIKey == "" || c.AppUserName == "" || c.AppUserPass == "" {
			return fmt.Errorf("LLM_API_KEY, APP_USER_NAME and APP_USER_PASS are required")
		}
		return nil
	case StoreS3:
		c.LLMAPIKey, c.AppUserName, c.AppUserPass = "", "", ""
	default:
		return fmt.Errorf("unknown STORE_BACKEND %q", c.StoreBackend)
	}
	if c.SSMKeyName == "" || c.SSMUserKeyName == "" || c.SSMPassKeyName == "" {
		return fmt.Errorf("SSM_KEY_NAME, SSM_USER_KEY_NAME and SSM_PASS_KEY_NAME are required for s3")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS configuration for secrets: %w", err)
	}
	return c.resolveSecrets(ctx, ssm.NewFromConfig(cfg))
}

func (c *Config) resolveSecrets(ctx context.Context, client parameterGetter) error {
	c.LLMAPIKey, c.AppUserName, c.AppUserPass = "", "", ""
	values := make([]string, 0, 3)
	for _, secret := range []struct{ label, name string }{
		{"LLM API key", c.SSMKeyName},
		{"app username", c.SSMUserKeyName},
		{"app password", c.SSMPassKeyName},
	} {
		result, err := client.GetParameter(ctx, &ssm.GetParameterInput{
			Name: aws.String(secret.name), WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("retrieve %s from SSM: %w", secret.label, err)
		}
		if result == nil || result.Parameter == nil || result.Parameter.Value == nil ||
			strings.TrimSpace(*result.Parameter.Value) == "" {
			return fmt.Errorf("SSM %s parameter has no value", secret.label)
		}
		values = append(values, strings.TrimSpace(*result.Parameter.Value))
	}
	if strings.Contains(values[1], ":") {
		return fmt.Errorf("SSM app username must not contain a colon")
	}
	c.LLMAPIKey, c.AppUserName, c.AppUserPass = values[0], values[1], values[2]
	return nil
}
