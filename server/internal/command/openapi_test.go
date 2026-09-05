package command

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/appkernia/appkernia/server/internal/platform/runtimeassets"
)

const testOpenAPI = `openapi: 3.1.0
components:
  parameters:
    Locale:
      name: Accept-Language
      in: header
      required: false
      schema: {type: string}
  schemas:
    ArticleRequest:
      type: object
      required: [title]
      properties: {title: {type: string}}
paths:
  /admin-api/v1/apps/{app_id}/articles:
    post:
      summary: Create article
      operationId: createArticle
      x-appkernia-agent-callable: true
      security: [{adminBearer: []}, {apiClientBearer: []}]
      parameters:
        - {name: app_id, in: path, required: true, schema: {type: string}}
        - {name: tag, in: query, schema: {type: string}}
        - {$ref: '#/components/parameters/Locale'}
      requestBody: {required: true, content: {application/json: {schema: {$ref: '#/components/schemas/ArticleRequest'}}}}
  /admin-api/v1/private:
    get:
      operationId: privateOperation
      x-appkernia-agent-callable: false
      security: [{adminBearer: []}]
`

func TestOpenAPIOperationPreparationIsAllowlisted(t *testing.T) {
	operations, err := parseAPIOperations([]byte(testOpenAPI))
	if err != nil {
		t.Fatal(err)
	}
	operation := operations["createArticle"]
	if !operation.AgentCallable || operations["privateOperation"].AgentCallable {
		t.Fatalf("unexpected callable operations: %#v", operations)
	}
	if schema, ok := operation.RequestBodySchema.(map[string]any); !ok || schema["type"] != "object" {
		t.Fatalf("request schema was not resolved: %#v", operation.RequestBodySchema)
	}
	request, err := prepareAPIRequest(operation,
		[]string{"app_id=app one"}, []string{"tag=a", "tag=b"}, []string{"Accept-Language=zh-CN"}, `{"title":"hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if request.Path != "/admin-api/v1/apps/app%20one/articles" || request.Query != "tag=a&tag=b" || request.Headers["Accept-Language"] != "zh-CN" {
		t.Fatalf("unexpected request: %#v", request)
	}
	if _, err = prepareAPIRequest(operation, nil, nil, nil, ""); err == nil {
		t.Fatal("missing required path and body were accepted")
	}
}

func TestClientTokenExchangeDoesNotRetry(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), "secret-value") || request.URL.Path != "/api/v1/auth/client-token" {
			t.Fatalf("unexpected token request: %s %s", request.URL.Path, body)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"data":{"access_token":"token-value","audience":"ak-api"}}`))
	}))
	t.Cleanup(server.Close)
	credentials := clientCredentials{Server: server.URL, ClientID: "ak_test", ClientSecret: "aks_secret-value-that-is-long-enough"}
	token, err := exchangeClientToken(t.Context(), server.Client(), credentials)
	if err != nil || token != "token-value" || requests != 1 {
		t.Fatalf("token=%q requests=%d err=%v", token, requests, err)
	}
}

func TestAPICallDoesNotFollowRedirects(t *testing.T) {
	redirected := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/auth/client-token":
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write([]byte(`{"data":{"access_token":"token-value","audience":"ak-api"}}`))
		case "/write":
			http.Redirect(response, request, "/redirected", http.StatusTemporaryRedirect)
		case "/redirected":
			redirected++
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("AK_SERVER_URL", server.URL)
	t.Setenv("AK_CLIENT_ID", "ak_test")
	t.Setenv("AK_CLIENT_SECRET", "aks_secret-value-that-is-long-enough")
	operation := apiOperation{OperationID: "write", Method: http.MethodPost, Path: "/write", AgentCallable: true}
	err := callAPIOperation("akone", map[string]apiOperation{"write": operation}, []string{"write"})
	if err == nil || redirected != 0 {
		t.Fatalf("redirected=%d err=%v", redirected, err)
	}
}

func TestEmbeddedOpenAPIExposesNativeAndDelegatedClientOperations(t *testing.T) {
	document, err := runtimeassets.OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	operations, err := parseAPIOperations(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, operationID := range []string{"submitApplicationNotification", "getAdminDashboardSummary", "createAdminAppContentArticle"} {
		if !operations[operationID].AgentCallable {
			t.Fatalf("operation %s is not available to the CLI", operationID)
		}
	}
}
