package awslambda

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/imposter-project/imposter-go/pkg/logger"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/imposter-project/imposter-go/internal/adapter"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/plugin"
)

// LambdaAdapter represents the AWS Lambda runtime adapter
type LambdaAdapter struct{}

// NewAdapter creates a new Lambda adapter instance
func NewAdapter() adapter.Adapter {
	return &LambdaAdapter{}
}

// Start begins the Lambda runtime
func (a *LambdaAdapter) Start() {
	lambda.Start(HandleLambdaRequest)
}

var (
	imposterConfig *config.ImposterConfig
	plugins        []plugin.Plugin
)

func init() {
	// Only execute Lambda initialization if we're running in Lambda mode
	if !adapter.IsLambda() {
		return
	}

	startTime := time.Now()
	defer func() {
		logger.Infof("startup completed in %v", time.Since(startTime))
	}()

	// For Lambda, default to /var/task/config if IMPOSTER_CONFIG_DIR is not set
	if os.Getenv("IMPOSTER_CONFIG_DIR") == "" {
		logger.Infoln("IMPOSTER_CONFIG_DIR not set, defaulting to /var/task/config")
		os.Setenv("IMPOSTER_CONFIG_DIR", "/var/task/config")
	}

	// Load configuration once during cold start
	imposterConfig, plugins = adapter.InitialiseImposter("")
}

// HandleLambdaRequest handles incoming Lambda requests and routes them to the appropriate handler.
func HandleLambdaRequest(req json.RawMessage) (interface{}, error) {
	var apiGatewayReq events.APIGatewayProxyRequest
	var lambdaFunctionURLReq events.LambdaFunctionURLRequest

	if err := json.Unmarshal(req, &apiGatewayReq); err == nil && apiGatewayReq.HTTPMethod != "" {
		return handleAPIGatewayProxyRequest(apiGatewayReq, plugins)
	} else if err := json.Unmarshal(req, &lambdaFunctionURLReq); err == nil && lambdaFunctionURLReq.RequestContext.HTTP.Method != "" {
		return handleLambdaFunctionURLRequest(lambdaFunctionURLReq, plugins)
	} else {
		return events.LambdaFunctionURLResponse{StatusCode: 400, Body: "Unsupported request type"}, nil
	}
}

// handleAPIGatewayProxyRequest processes API Gateway Proxy requests.
func handleAPIGatewayProxyRequest(req events.APIGatewayProxyRequest, plugins []plugin.Plugin) (events.APIGatewayProxyResponse, error) {
	// Convert APIGatewayProxyRequest to http.Request
	httpReq, err := convertLambdaRequestToHTTPRequest(req.HTTPMethod, req.Path, req.Headers, req.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Failed to convert request"}, nil
	}
	logRequest(httpReq)

	// Create a responseRecorder to capture the response
	recorder := &responseRecorder{Headers: make(http.Header)}

	// Handle the request
	handler.HandleRequest(imposterConfig, recorder, httpReq, plugins)
	logResponse(recorder)

	// Convert the captured response to APIGatewayProxyResponse
	return convertHTTPResponseToLambdaResponse(recorder), nil
}

// handleLambdaFunctionURLRequest processes Lambda Function URL requests.
func handleLambdaFunctionURLRequest(req events.LambdaFunctionURLRequest, plugins []plugin.Plugin) (events.LambdaFunctionURLResponse, error) {
	// Convert LambdaFunctionURLRequest to http.Request
	httpReq, err := convertLambdaRequestToHTTPRequest(req.RequestContext.HTTP.Method, req.RawPath, req.Headers, req.Body)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: 500, Body: "Failed to convert request"}, nil
	}
	logRequest(httpReq)

	// Create a responseRecorder to capture the response
	recorder := &responseRecorder{Headers: make(http.Header)}

	// Handle the request
	handler.HandleRequest(imposterConfig, recorder, httpReq, plugins)
	logResponse(recorder)

	// Convert the captured response to LambdaFunctionURLResponse
	return convertHTTPResponseToLambdaFunctionURLResponse(recorder), nil
}

// convertLambdaRequestToHTTPRequest converts a Lambda request to an http.Request.
func convertLambdaRequestToHTTPRequest(method, path string, headers map[string]string, body string) (*http.Request, error) {
	bodyReader := strings.NewReader(body)
	httpReq, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	return httpReq, nil
}

// convertHTTPResponseToLambdaResponse converts an http.Response to an APIGatewayProxyResponse.
// Binary responses are base64-encoded so API Gateway can deliver them correctly.
func convertHTTPResponseToLambdaResponse(recorder *responseRecorder) events.APIGatewayProxyResponse {
	resp := events.APIGatewayProxyResponse{
		StatusCode: recorder.StatusCode,
		Headers:    convertHTTPHeaderToMap(recorder.Headers),
	}
	if isBinaryResponse(recorder) {
		resp.Body = base64.StdEncoding.EncodeToString(recorder.Body.Bytes())
		resp.IsBase64Encoded = true
	} else {
		resp.Body = recorder.Body.String()
	}
	return resp
}

// convertHTTPResponseToLambdaFunctionURLResponse converts an http.Response to a LambdaFunctionURLResponse.
// Binary responses are base64-encoded so Lambda Function URLs can deliver them correctly.
func convertHTTPResponseToLambdaFunctionURLResponse(recorder *responseRecorder) events.LambdaFunctionURLResponse {
	resp := events.LambdaFunctionURLResponse{
		StatusCode: recorder.StatusCode,
		Headers:    convertHTTPHeaderToMap(recorder.Headers),
	}
	if isBinaryResponse(recorder) {
		resp.Body = base64.StdEncoding.EncodeToString(recorder.Body.Bytes())
		resp.IsBase64Encoded = true
	} else {
		resp.Body = recorder.Body.String()
	}
	return resp
}

// isBinaryResponse returns true if the response contains binary data
// that must be base64-encoded for Lambda delivery. Known textual content
// types are fast-pathed; otherwise the body is classified as binary if it
// is not valid UTF-8 or contains a NUL byte. NUL never appears in legitimate
// text payloads and catches binary formats (e.g. gRPC frames, images,
// protobuf) whose bytes may otherwise be valid UTF-8 by coincidence.
func isBinaryResponse(recorder *responseRecorder) bool {
	contentType := recorder.Headers.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/") ||
		strings.HasSuffix(contentType, "+json") ||
		strings.HasSuffix(contentType, "+xml") {
		return false
	}
	body := recorder.Body.Bytes()
	return !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0
}

// convertHTTPHeaderToMap converts http.Header to a map[string]string.
func convertHTTPHeaderToMap(header http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range header {
		result[key] = strings.Join(values, ",")
	}
	return result
}

// logRequest logs the incoming HTTP request at TRACE level
func logRequest(req *http.Request) {
	logger.Tracef("request: %s %s", req.Method, req.URL.String())
}

// logResponse logs the outgoing HTTP response at TRACE level
func logResponse(resp *responseRecorder) {
	logger.Tracef("response: %d %s", resp.StatusCode, &resp.Body)
}
