package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"go-chat/app/llm"
	"go-chat/app/store"

	"github.com/aws/aws-lambda-go/events"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

func TestLambdaFunctionURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	model, err := llm.NewFake("test-key", "test-model")
	if err != nil {
		t.Fatal(err)
	}
	adapter := ginadapter.NewV2(newRouter(store.NewMemory(), model, "user", "password"))
	invoke := func(method, path, body string, authenticated bool) events.APIGatewayV2HTTPResponse {
		t.Helper()
		headers := map[string]string{
			"content-type": "application/json",
			"origin":       "https://example.lambda-url.us-east-1.on.aws",
		}
		if authenticated {
			headers["authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte("user:password"))
		}
		response, err := adapter.ProxyWithContext(context.Background(), events.APIGatewayV2HTTPRequest{
			Version: "2.0",
			RawPath: path,
			Headers: headers,
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				DomainName: "example.lambda-url.us-east-1.on.aws",
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
					Method: method,
					Path:   path,
				},
			},
			Body:            base64.StdEncoding.EncodeToString([]byte(body)),
			IsBase64Encoded: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	for _, path := range []string{"/", "/config.js", "/history", "/healthz", "/chat"} {
		method := http.MethodGet
		if path == "/chat" {
			method = http.MethodPost
		}
		response := invoke(method, path, `{"prompt":"hello"}`, false)
		if response.StatusCode != http.StatusUnauthorized || response.Headers[http.CanonicalHeaderKey("WWW-Authenticate")] == "" {
			t.Fatalf("%s: expected Basic Auth challenge, got %+v", path, response)
		}
		response = invoke(method, path, `{"prompt":"hello"}`, true)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %+v", path, response)
		}
	}
	response := invoke(http.MethodGet, "/history", "", true)
	body := []byte(response.Body)
	if response.IsBase64Encoded {
		body, err = base64.StdEncoding.DecodeString(response.Body)
		if err != nil {
			t.Fatal(err)
		}
	}
	var history []store.Exchange
	if err := json.Unmarshal(body, &history); err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Prompt != "hello" || history[0].Response == "" {
		t.Fatalf("chat was not persisted: %+v", history)
	}
}
