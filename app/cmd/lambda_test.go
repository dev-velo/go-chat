package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"go-chat/app/llm"
	"go-chat/app/store"

	"github.com/aws/aws-lambda-go/events"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

const functionURLHost = "example.lambda-url.us-east-1.on.aws"

// TestLambdaFunctionURL drives the app through the payload shape a function
// URL actually uses, which is where Basic Auth broke: the URL rewrites a
// WWW-Authenticate header to x-amzn-Remapped-www-authenticate, so the browser
// never saw the challenge. The session cookie avoids that, but only if it
// leaves in the response's Cookies array — a Set-Cookie header would be
// rewritten the same way.
func TestLambdaFunctionURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	model, err := llm.NewFake("test-key", "test-model")
	if err != nil {
		t.Fatal(err)
	}
	adapter := ginadapter.NewV2(newRouter(store.NewMemory(), model, "user", "password"))

	invoke := func(method, path, body string, headers map[string]string, cookies []string) events.APIGatewayV2HTTPResponse {
		t.Helper()
		all := map[string]string{"origin": "https://" + functionURLHost}
		for key, value := range headers {
			all[key] = value
		}
		response, err := adapter.ProxyWithContext(context.Background(), events.APIGatewayV2HTTPRequest{
			Version: "2.0",
			RawPath: path,
			Headers: all,
			Cookies: cookies,
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				DomainName: functionURLHost,
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

	jsonHeaders := map[string]string{"content-type": "application/json"}
	for _, path := range []string{"/", "/config.js", "/history", "/healthz", "/chat"} {
		method := http.MethodGet
		if path == "/chat" {
			method = http.MethodPost
		}
		response := invoke(method, path, `{"prompt":"hello"}`, jsonHeaders, nil)
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401 without a session, got %d", path, response.StatusCode)
		}
	}

	// A browser navigating in gets the form rather than a dead challenge.
	response := invoke(http.MethodGet, "/", "", map[string]string{"accept": "text/html"}, nil)
	if response.StatusCode != http.StatusSeeOther ||
		!strings.HasPrefix(response.Headers[http.CanonicalHeaderKey("Location")], "/login") {
		t.Fatalf("expected a redirect to the login form, got %+v", response)
	}

	form := url.Values{"username": {"user"}, "password": {"password"}}.Encode()
	response = invoke(http.MethodPost, "/login", form,
		map[string]string{"content-type": "application/x-www-form-urlencoded"}, nil)
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("login: expected 303, got %+v", response)
	}
	if len(response.Cookies) == 0 {
		t.Fatal("login returned no cookies: a function URL only emits Set-Cookie from the Cookies array")
	}
	for name := range response.Headers {
		if strings.EqualFold(name, "Set-Cookie") {
			t.Fatal("Set-Cookie was sent as a header, which a function URL rewrites")
		}
	}
	session := response.Cookies[0]
	if !strings.Contains(session, "HttpOnly") || !strings.Contains(session, "Secure") ||
		!strings.Contains(session, "SameSite=Lax") {
		t.Fatalf("weak cookie attributes over a function URL: %q", session)
	}

	// The cookie arrives back in the request's Cookies array, not a header.
	credentials := []string{strings.SplitN(session, ";", 2)[0]}
	for _, path := range []string{"/", "/config.js", "/history", "/healthz", "/chat"} {
		method := http.MethodGet
		if path == "/chat" {
			method = http.MethodPost
		}
		response := invoke(method, path, `{"prompt":"hello"}`, jsonHeaders, credentials)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200 with a session, got %+v", path, response)
		}
	}

	response = invoke(http.MethodGet, "/history", "", jsonHeaders, credentials)
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
