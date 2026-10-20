package config

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type stubParameterGetter struct {
	input  *ssm.GetParameterInput
	output *ssm.GetParameterOutput
	err    error
	ctx    context.Context
}

func (s *stubParameterGetter) GetParameter(ctx context.Context, input *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	s.input = input
	s.ctx = ctx
	return s.output, s.err
}

func TestResolveAPIKeyFromSSM(t *testing.T) {
	cfg := Config{SSMKeyName: "/go-chat/dev/llm-api-key"}
	client := &stubParameterGetter{output: &ssm.GetParameterOutput{
		Parameter: &types.Parameter{Value: aws.String(" test-key ")},
	}}
	ctx := context.Background()
	if err := cfg.resolveAPIKey(ctx, client); err != nil {
		t.Fatal(err)
	}
	if cfg.LLMAPIKey != "test-key" {
		t.Fatal("expected resolved and trimmed key")
	}
	if aws.ToString(client.input.Name) != cfg.SSMKeyName || !aws.ToBool(client.input.WithDecryption) {
		t.Fatal("expected requested parameter with decryption enabled")
	}
	if client.ctx != ctx {
		t.Fatal("expected caller context to reach the SDK")
	}
}

func TestResolveAPIKeyRejectsEmptySSMValue(t *testing.T) {
	for _, output := range []*ssm.GetParameterOutput{
		nil,
		{},
		{Parameter: &types.Parameter{}},
		{Parameter: &types.Parameter{Value: aws.String(" \t ")}},
	} {
		cfg := Config{SSMKeyName: "/go-chat/dev/llm-api-key"}
		err := cfg.resolveAPIKey(context.Background(), &stubParameterGetter{output: output})
		if err == nil || cfg.LLMAPIKey != "" {
			t.Fatal("expected empty SSM value to fail without setting a key")
		}
	}
}

func TestResolveAPIKeyPropagatesSSMError(t *testing.T) {
	for _, cause := range []error{errors.New("access denied"), context.DeadlineExceeded} {
		cfg := Config{SSMKeyName: "/go-chat/dev/llm-api-key"}
		err := cfg.resolveAPIKey(context.Background(), &stubParameterGetter{err: cause})
		if !errors.Is(err, cause) || cfg.LLMAPIKey != "" {
			t.Fatal("expected SSM failure to propagate without setting a key")
		}
	}
}

func TestResolveAPIKeyDirectEnvironmentNeedsNoAWS(t *testing.T) {
	cfg := Config{LLMAPIKey: "local-key", SSMKeyName: "/ignored"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cfg.ResolveAPIKey(ctx); err != nil {
		t.Fatal(err)
	}
	if cfg.LLMAPIKey != "local-key" {
		t.Fatal("expected local key unchanged")
	}
}

func TestResolveAPIKeyRejectsInvalidSources(t *testing.T) {
	for _, cfg := range []Config{{}, {LLMAPIKey: "local-key", SSMFetchKey: true}} {
		if err := cfg.ResolveAPIKey(context.Background()); err == nil {
			t.Fatal("expected invalid key sources to fail")
		}
	}
}
