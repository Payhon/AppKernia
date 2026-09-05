package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/appkernia/appkernia/server/internal/platform/buildinfo"
	"github.com/appkernia/appkernia/server/internal/platform/runtimeassets"
	"go.yaml.in/yaml/v3"
)

const maxAPIResponseBytes = 32 << 20

type apiParameter struct {
	Name     string `json:"name" yaml:"name"`
	In       string `json:"in" yaml:"in"`
	Required bool   `json:"required" yaml:"required"`
	Schema   any    `json:"schema,omitempty" yaml:"schema"`
	Ref      string `json:"-" yaml:"$ref"`
}

type apiOperation struct {
	OperationID         string         `json:"operation_id"`
	Method              string         `json:"method"`
	Path                string         `json:"path"`
	Summary             string         `json:"summary"`
	Parameters          []apiParameter `json:"parameters"`
	RequestBodyRequired bool           `json:"request_body_required"`
	RequestBody         bool           `json:"request_body"`
	RequestBodySchema   any            `json:"request_body_schema,omitempty"`
	AgentCallable       bool           `json:"agent_callable"`
}

type openAPISpec struct {
	Paths      map[string]openAPIPathItem `yaml:"paths"`
	Components struct {
		Parameters map[string]apiParameter `yaml:"parameters"`
		Schemas    map[string]any          `yaml:"schemas"`
	} `yaml:"components"`
}

type openAPIPathItem struct {
	Parameters []apiParameter       `yaml:"parameters"`
	Delete     *openAPIRawOperation `yaml:"delete"`
	Get        *openAPIRawOperation `yaml:"get"`
	Head       *openAPIRawOperation `yaml:"head"`
	Patch      *openAPIRawOperation `yaml:"patch"`
	Post       *openAPIRawOperation `yaml:"post"`
	Put        *openAPIRawOperation `yaml:"put"`
}

type openAPIRawOperation struct {
	OperationID   string           `yaml:"operationId"`
	Summary       string           `yaml:"summary"`
	AgentCallable bool             `yaml:"x-appkernia-agent-callable"`
	Security      []map[string]any `yaml:"security"`
	Parameters    []apiParameter   `yaml:"parameters"`
	RequestBody   *struct {
		Required bool `yaml:"required"`
		Content  map[string]struct {
			Schema any `yaml:"schema"`
		} `yaml:"content"`
	} `yaml:"requestBody"`
}

func apiCommand(program string, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(os.Stdout, "usage: %s api list|describe|call\n", program)
		return nil
	}
	if len(args) == 0 {
		return &UsageError{Message: fmt.Sprintf("usage: %s api list|describe|call", program)}
	}
	specification, err := runtimeassets.OpenAPI()
	if err != nil {
		return err
	}
	operations, err := parseAPIOperations(specification)
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprintf(os.Stdout, "usage: %s api list\n", program)
			return nil
		}
		if len(args) != 1 {
			return &UsageError{Message: fmt.Sprintf("usage: %s api list", program)}
		}
		callable := make([]apiOperation, 0, len(operations))
		operationIDs := make([]string, 0, len(operations))
		for operationID := range operations {
			operationIDs = append(operationIDs, operationID)
		}
		sort.Strings(operationIDs)
		for _, operationID := range operationIDs {
			operation := operations[operationID]
			if operation.AgentCallable {
				operation.Parameters = nil
				operation.RequestBodySchema = nil
				callable = append(callable, operation)
			}
		}
		return json.NewEncoder(os.Stdout).Encode(callable)
	case "describe":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprintf(os.Stdout, "usage: %s api describe OPERATION_ID\n", program)
			return nil
		}
		if len(args) != 2 {
			return &UsageError{Message: fmt.Sprintf("usage: %s api describe OPERATION_ID", program)}
		}
		operation, ok := operations[args[1]]
		if !ok || !operation.AgentCallable {
			return fmt.Errorf("agent-callable OpenAPI operation %q was not found", args[1])
		}
		return json.NewEncoder(os.Stdout).Encode(operation)
	case "call":
		return callAPIOperation(program, operations, args[1:])
	default:
		return &UsageError{Message: fmt.Sprintf("usage: %s api list|describe|call", program)}
	}
}

func parseAPIOperations(document []byte) (map[string]apiOperation, error) {
	var specification openAPISpec
	decoder := yaml.NewDecoder(bytes.NewReader(document))
	decoder.KnownFields(false)
	if err := decoder.Decode(&specification); err != nil {
		return nil, fmt.Errorf("decode embedded OpenAPI: %w", err)
	}
	operations := make(map[string]apiOperation)
	paths := make([]string, 0, len(specification.Paths))
	for path := range specification.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, routePath := range paths {
		pathItem := specification.Paths[routePath]
		methods := []struct {
			name string
			raw  *openAPIRawOperation
		}{{"DELETE", pathItem.Delete}, {"GET", pathItem.Get}, {"HEAD", pathItem.Head}, {"PATCH", pathItem.Patch}, {"POST", pathItem.Post}, {"PUT", pathItem.Put}}
		for _, candidate := range methods {
			if candidate.raw == nil || strings.TrimSpace(candidate.raw.OperationID) == "" {
				continue
			}
			raw := *candidate.raw
			parameters := make([]apiParameter, 0, len(pathItem.Parameters)+len(raw.Parameters))
			for _, parameter := range append(append([]apiParameter(nil), pathItem.Parameters...), raw.Parameters...) {
				if parameter.Ref != "" {
					const prefix = "#/components/parameters/"
					resolved, ok := specification.Components.Parameters[strings.TrimPrefix(parameter.Ref, prefix)]
					if !strings.HasPrefix(parameter.Ref, prefix) || !ok {
						return nil, fmt.Errorf("operation %s contains an unresolved parameter reference", raw.OperationID)
					}
					parameter = resolved
				}
				parameters = append(parameters, parameter)
			}
			for index := range parameters {
				parameters[index].Schema = resolveSchema(parameters[index].Schema, specification.Components.Schemas, map[string]bool{})
			}
			callable := hasAPIClientSecurity(raw.Security)
			if raw.AgentCallable && !callable {
				return nil, fmt.Errorf("agent-callable operation %s is missing apiClientBearer security", raw.OperationID)
			}
			operation := apiOperation{OperationID: raw.OperationID, Method: candidate.name, Path: routePath, Summary: raw.Summary, Parameters: parameters, AgentCallable: callable}
			if raw.RequestBody != nil {
				operation.RequestBody = true
				operation.RequestBodyRequired = raw.RequestBody.Required
				if media, ok := raw.RequestBody.Content["application/json"]; ok {
					operation.RequestBodySchema = resolveSchema(media.Schema, specification.Components.Schemas, map[string]bool{})
				}
			}
			if _, exists := operations[operation.OperationID]; exists {
				return nil, fmt.Errorf("OpenAPI operationId %q is duplicated", operation.OperationID)
			}
			operations[operation.OperationID] = operation
		}
	}
	return operations, nil
}

func resolveSchema(value any, components map[string]any, visiting map[string]bool) any {
	if object, ok := value.(map[string]any); ok {
		if reference, ok := object["$ref"].(string); ok {
			const prefix = "#/components/schemas/"
			name := strings.TrimPrefix(reference, prefix)
			component, exists := components[name]
			if strings.HasPrefix(reference, prefix) && exists && !visiting[name] {
				visiting[name] = true
				resolved := resolveSchema(component, components, visiting)
				delete(visiting, name)
				return resolved
			}
		}
		resolved := make(map[string]any, len(object))
		for key, child := range object {
			resolved[key] = resolveSchema(child, components, visiting)
		}
		return resolved
	}
	if items, ok := value.([]any); ok {
		resolved := make([]any, len(items))
		for index, child := range items {
			resolved[index] = resolveSchema(child, components, visiting)
		}
		return resolved
	}
	return value
}

func hasAPIClientSecurity(security []map[string]any) bool {
	for _, alternative := range security {
		if _, ok := alternative["apiClientBearer"]; ok {
			return true
		}
	}
	return false
}

type stringFlags []string

func (values *stringFlags) String() string { return strings.Join(*values, ",") }
func (values *stringFlags) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func callAPIOperation(program string, operations map[string]apiOperation, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(os.Stdout, "usage: %s api call OPERATION_ID [--path name=value] [--query name=value] [--header name=value] [--body @file|-]\n", program)
		return nil
	}
	if len(args) == 0 {
		return &UsageError{Message: fmt.Sprintf("usage: %s api call OPERATION_ID [--path name=value] [--query name=value] [--header name=value] [--body @file|-]", program)}
	}
	operation, ok := operations[args[0]]
	if !ok || !operation.AgentCallable {
		return fmt.Errorf("agent-callable OpenAPI operation %q was not found", args[0])
	}
	flags := flag.NewFlagSet("api call", flag.ContinueOnError)
	var pathValues, queryValues, headerValues stringFlags
	flags.Var(&pathValues, "path", "path parameter name=value (repeatable)")
	flags.Var(&queryValues, "query", "query parameter name=value (repeatable)")
	flags.Var(&headerValues, "header", "header parameter name=value (repeatable)")
	bodyValue := flags.String("body", "", "JSON body, @file, or - for stdin")
	server := flags.String("server", "", "override AppKernia server origin")
	credentialsFile := flags.String("credentials-file", "", "credentials file path")
	timeout := flags.Duration("timeout", 30*time.Second, "request timeout")
	if err := parseCommandFlags(flags, args[1:], fmt.Sprintf("usage: %s api call OPERATION_ID [--path name=value] [--query name=value] [--header name=value] [--body @file|-]", program)); err != nil {
		return err
	}
	if *timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	request, err := prepareAPIRequest(operation, pathValues, queryValues, headerValues, *bodyValue)
	if err != nil {
		return err
	}
	credentials, err := loadCredentials(*credentialsFile, *server)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	token, err := exchangeClientToken(ctx, client, credentials)
	if err != nil {
		return err
	}
	target := credentials.Server + request.Path
	if request.Query != "" {
		target += "?" + request.Query
	}
	httpRequest, err := http.NewRequestWithContext(ctx, operation.Method, target, bytes.NewReader(request.Body))
	if err != nil {
		return fmt.Errorf("create API request: %w", err)
	}
	for name, value := range request.Headers {
		httpRequest.Header.Set(name, value)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+token)
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("User-Agent", "akone/"+buildinfo.Version)
	if len(request.Body) > 0 {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("call %s: %w", operation.OperationID, err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, maxAPIResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read API response: %w", err)
	}
	if len(content) > maxAPIResponseBytes {
		return errors.New("API response exceeds 32 MiB")
	}
	if len(content) > 0 {
		if _, err = os.Stdout.Write(content); err != nil {
			return err
		}
		if content[len(content)-1] != '\n' {
			fmt.Fprintln(os.Stdout)
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("API operation %s returned HTTP %d", operation.OperationID, response.StatusCode)
	}
	return nil
}

type preparedAPIRequest struct {
	Path    string
	Query   string
	Headers map[string]string
	Body    []byte
}

func prepareAPIRequest(operation apiOperation, pathFlags, queryFlags, headerFlags []string, bodyValue string) (preparedAPIRequest, error) {
	allowed := map[string]map[string]apiParameter{"path": {}, "query": {}, "header": {}}
	for _, parameter := range operation.Parameters {
		if values := allowed[parameter.In]; values != nil {
			values[strings.ToLower(parameter.Name)] = parameter
		}
	}
	pathValues, err := parseNamedValues(pathFlags, allowed["path"], false)
	if err != nil {
		return preparedAPIRequest{}, err
	}
	queryValues, err := parseNamedValues(queryFlags, allowed["query"], true)
	if err != nil {
		return preparedAPIRequest{}, err
	}
	headers, err := parseNamedValues(headerFlags, allowed["header"], false)
	if err != nil {
		return preparedAPIRequest{}, err
	}
	pathValue := operation.Path
	for _, parameter := range operation.Parameters {
		if parameter.In != "path" {
			continue
		}
		values := pathValues[strings.ToLower(parameter.Name)]
		if parameter.Required && len(values) == 0 {
			return preparedAPIRequest{}, fmt.Errorf("missing required path parameter %s", parameter.Name)
		}
		if len(values) > 0 {
			pathValue = strings.ReplaceAll(pathValue, "{"+parameter.Name+"}", url.PathEscape(values[0]))
		}
	}
	if strings.Contains(pathValue, "{") {
		return preparedAPIRequest{}, errors.New("not all path parameters were supplied")
	}
	for _, parameter := range operation.Parameters {
		if !parameter.Required || parameter.In == "path" {
			continue
		}
		values := map[string][]string(nil)
		if parameter.In == "query" {
			values = queryValues
		} else if parameter.In == "header" {
			values = headers
		}
		if values != nil && len(values[strings.ToLower(parameter.Name)]) == 0 {
			return preparedAPIRequest{}, fmt.Errorf("missing required %s parameter %s", parameter.In, parameter.Name)
		}
	}
	query := url.Values{}
	for key, values := range queryValues {
		name := allowed["query"][key].Name
		for _, value := range values {
			query.Add(name, value)
		}
	}
	header := make(map[string]string, len(headers))
	for key, values := range headers {
		name := allowed["header"][key].Name
		if strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "Host") || strings.EqualFold(name, "Content-Length") {
			return preparedAPIRequest{}, fmt.Errorf("header %s is managed by akone", name)
		}
		header[name] = values[0]
	}
	body, err := readRequestBody(bodyValue)
	if err != nil {
		return preparedAPIRequest{}, err
	}
	if operation.RequestBodyRequired && len(body) == 0 {
		return preparedAPIRequest{}, errors.New("operation requires --body")
	}
	if !operation.RequestBody && len(body) > 0 {
		return preparedAPIRequest{}, errors.New("operation does not accept --body")
	}
	if len(body) > 0 && !json.Valid(body) {
		return preparedAPIRequest{}, errors.New("request body must be valid JSON")
	}
	return preparedAPIRequest{Path: pathValue, Query: query.Encode(), Headers: header, Body: body}, nil
}

func parseNamedValues(flags []string, allowed map[string]apiParameter, repeated bool) (map[string][]string, error) {
	values := make(map[string][]string, len(flags))
	for _, raw := range flags {
		name, value, ok := strings.Cut(raw, "=")
		key := strings.ToLower(strings.TrimSpace(name))
		if !ok || key == "" {
			return nil, fmt.Errorf("parameter %q must use name=value", raw)
		}
		if _, ok = allowed[key]; !ok {
			return nil, fmt.Errorf("parameter %q is not declared by the OpenAPI operation", name)
		}
		if !repeated && len(values[key]) > 0 {
			return nil, fmt.Errorf("parameter %q may be supplied only once", name)
		}
		values[key] = append(values[key], value)
	}
	return values, nil
}

func readRequestBody(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	var reader io.Reader = strings.NewReader(value)
	if value == "-" {
		reader = os.Stdin
	} else if strings.HasPrefix(value, "@") {
		file, err := os.Open(strings.TrimPrefix(value, "@"))
		if err != nil {
			return nil, fmt.Errorf("open request body: %w", err)
		}
		defer file.Close()
		reader = file
	}
	body, err := io.ReadAll(io.LimitReader(reader, maxAPIResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if len(body) > maxAPIResponseBytes {
		return nil, errors.New("request body exceeds 32 MiB")
	}
	return body, nil
}

func exchangeClientToken(ctx context.Context, client *http.Client, credentials clientCredentials) (string, error) {
	payload, _ := json.Marshal(map[string]string{"client_id": credentials.ClientID, "client_secret": credentials.ClientSecret})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, credentials.Server+"/api/v1/auth/client-token", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create client-token request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "akone/"+buildinfo.Version)
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("exchange client secret: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read client-token response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("client-token exchange returned HTTP %d", response.StatusCode)
	}
	var envelope struct {
		Data struct {
			AccessToken string `json:"access_token"`
			Audience    string `json:"audience"`
		} `json:"data"`
	}
	if err = json.Unmarshal(body, &envelope); err != nil || strings.TrimSpace(envelope.Data.AccessToken) == "" || envelope.Data.Audience != "ak-api" {
		return "", errors.New("client-token response is invalid")
	}
	return envelope.Data.AccessToken, nil
}
