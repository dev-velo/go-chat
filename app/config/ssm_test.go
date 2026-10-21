package config

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type stubParameterGetter func(context.Context, *ssm.GetParameterInput) (*ssm.GetParameterOutput, error)

func (s stubParameterGetter) GetParameter(ctx context.Context, in *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return s(ctx, in)
}

func TestResolveSecretsSSM(t *testing.T) {
	cfg := Config{StoreBackend: StoreS3, SSMKeyName: "/llm", SSMUserKeyName: "/user", SSMPassKeyName: "/pass"}
	values := map[string]string{"/llm": "key", "/user": "user", "/pass": "password"}
	calls := 0
	err := cfg.resolveSecrets(context.Background(), stubParameterGetter(func(_ context.Context, in *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
		calls++
		if !aws.ToBool(in.WithDecryption) {
			t.Fatal("decryption required")
		}
		return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: aws.String(values[aws.ToString(in.Name)])}}, nil
	}))
	if err != nil || calls != 3 || cfg.LLMAPIKey != "key" || cfg.AppUserName != "user" || cfg.AppUserPass != "password" {
		t.Fatalf("expected all secrets resolved: %v", err)
	}
}

func TestResolveSecretsFailsClosed(t *testing.T) {
	cause := errors.New("access denied")
	for _, failAt := range []int{1, 2, 3} {
		for _, empty := range []bool{true, false} {
			cfg := Config{LLMAPIKey: "fallback", AppUserName: "fallback", AppUserPass: "fallback"}
			calls := 0
			err := cfg.resolveSecrets(context.Background(), stubParameterGetter(func(context.Context, *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
				calls++
				if calls == failAt {
					if empty {
						return &ssm.GetParameterOutput{}, nil
					}
					return nil, cause
				}
				return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: aws.String("value")}}, nil
			}))
			if err == nil || (!empty && !errors.Is(err, cause)) || cfg.LLMAPIKey != "" || cfg.AppUserName != "" || cfg.AppUserPass != "" {
				t.Fatal("failure must propagate without publishing partial or fallback secrets")
			}
		}
	}
}

func TestResolveSecretsMemoryNeedsNoAWS(t *testing.T) {
	cfg := Config{StoreBackend: StoreMemory, LLMAPIKey: "key", AppUserName: "user", AppUserPass: "password"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cfg.ResolveSecrets(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestResolveSecretsRejectsMissingConfiguration(t *testing.T) {
	for _, cfg := range []Config{{}, {StoreBackend: StoreMemory}, {StoreBackend: StoreS3}} {
		if err := cfg.ResolveSecrets(context.Background()); err == nil {
			t.Fatal("missing configuration must fail")
		}
	}
}

func TestResolveSecretsContextAndInvalidValues(t *testing.T) {
	for _, value := range []string{" \t ", "invalid:user"} {
		cfg := Config{SSMKeyName: "/llm", SSMUserKeyName: "/user", SSMPassKeyName: "/pass"}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := cfg.resolveSecrets(ctx, stubParameterGetter(func(got context.Context, in *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
			if got != ctx || got.Err() != context.Canceled {
				t.Fatal("caller context must reach SDK")
			}
			v := "valid"
			if aws.ToString(in.Name) == "/user" {
				v = value
			}
			return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: &v}}, nil
		}))
		if err == nil || cfg.AppUserName != "" || cfg.AppUserPass != "" || cfg.LLMAPIKey != "" {
			t.Fatal("invalid values must fail without publishing credentials")
		}
	}
}
