package awslambda

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
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
func (a *LambdaAdapter) Start() error {
	lambda.Start(HandleLambdaRequest)
	return nil
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

	// Load configuration once during cold start. A failure here is fatal:
	// panicking is the AWS convention for signalling a failed init phase, and
	// the error now carries a concise, user-facing message.
	var err error
	imposterConfig, plugins, err = adapter.InitialiseImposter("")
	if err != nil {
		logger.Errorf("failed to initialise imposter: %v", err)
		panic(err)
	}

	// Schedules and websocket connections require a long-lived process, which
	// the Lambda execution model does not provide.
	for _, plg := range plugins {
		cfg := plg.GetConfig()
		if len(cfg.Schedules) > 0 {
			logger.Warnf("schedules are not supported in Lambda and will not run")
		}
		if cfg.Plugin == "websocket" {
			logger.Warnf("the websocket plugin is not supported in Lambda")
		}
	}
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
	// Decode the body first: API Gateway base64-encodes binary payloads (e.g.
	// file uploads and multipart bodies) and sets IsBase64Encoded accordingly.
	body, err := decodeLambdaBody(req.Body, req.IsBase64Encoded)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Failed to decode request body"}, nil
	}

	// Convert APIGatewayProxyRequest to http.Request, preserving query parameters
	// and headers so that route matching and script access (queryParams, headers,
	// formParams derived from the body) work.
	httpReq, err := convertLambdaRequestToHTTPRequest(req.HTTPMethod, req.Path, buildAPIGatewayQueryString(req), buildAPIGatewayHeaders(req), body)
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
	// Decode the body first: the v2 payload base64-encodes binary payloads (e.g.
	// file uploads and multipart bodies) and sets IsBase64Encoded accordingly.
	body, err := decodeLambdaBody(req.Body, req.IsBase64Encoded)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: 400, Body: "Failed to decode request body"}, nil
	}

	// Convert LambdaFunctionURLRequest to http.Request. RawQueryString is already
	// URL-encoded in the v2 payload, so it can be used directly as the raw query.
	// Cookies arrive in a separate array (not the headers map) and are folded
	// back into a Cookie header by buildFunctionURLHeaders.
	httpReq, err := convertLambdaRequestToHTTPRequest(req.RequestContext.HTTP.Method, req.RawPath, req.RawQueryString, buildFunctionURLHeaders(req), body)
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
// rawQuery is the URL-encoded query string (without the leading '?') and is set
// on the request URL so downstream matching and scripts can read query
// parameters. body is the already-decoded request body; header carries the
// request headers (including any recombined cookies).
func convertLambdaRequestToHTTPRequest(method, path, rawQuery string, header http.Header, body []byte) (*http.Request, error) {
	httpReq, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if rawQuery != "" {
		httpReq.URL.RawQuery = rawQuery
	}
	if header != nil {
		httpReq.Header = header
	}

	return httpReq, nil
}

// decodeLambdaBody returns the request body bytes, base64-decoding them when the
// Lambda event marks the body as base64-encoded. API Gateway and Function URLs
// base64-encode binary request payloads (e.g. file uploads, multipart bodies,
// non-UTF-8 content); passing the raw base64 string downstream would corrupt
// body matching and form parsing.
func decodeLambdaBody(body string, isBase64 bool) ([]byte, error) {
	if !isBase64 {
		return []byte(body), nil
	}
	return base64.StdEncoding.DecodeString(body)
}

// buildAPIGatewayQueryString reconstructs an encoded query string from an API
// Gateway proxy (v1) request. Parameter values in the event are already
// URL-decoded, so they are re-encoded here. MultiValueQueryStringParameters is
// preferred when present because it preserves repeated keys (e.g. ?foo=a&foo=b);
// otherwise the single-valued QueryStringParameters map is used.
func buildAPIGatewayQueryString(req events.APIGatewayProxyRequest) string {
	values := url.Values{}
	if len(req.MultiValueQueryStringParameters) > 0 {
		for key, vals := range req.MultiValueQueryStringParameters {
			for _, v := range vals {
				values.Add(key, v)
			}
		}
	} else {
		for key, v := range req.QueryStringParameters {
			values.Set(key, v)
		}
	}
	return values.Encode()
}

// buildAPIGatewayHeaders converts the request headers of an API Gateway proxy
// (v1) event to http.Header. MultiValueHeaders is preferred when present so
// repeated headers (e.g. multiple Cookie or Accept values) are preserved;
// otherwise the single-valued Headers map is used.
func buildAPIGatewayHeaders(req events.APIGatewayProxyRequest) http.Header {
	header := make(http.Header)
	if len(req.MultiValueHeaders) > 0 {
		for key, vals := range req.MultiValueHeaders {
			for _, v := range vals {
				header.Add(key, v)
			}
		}
	} else {
		for key, v := range req.Headers {
			header.Set(key, v)
		}
	}
	return header
}

// buildFunctionURLHeaders converts the request headers of a Lambda Function URL
// / API Gateway HTTP API (v2) event to http.Header. In the v2 payload format
// cookies are delivered in a separate Cookies array rather than the headers
// map, so they are recombined into a single Cookie header (RFC 6265).
func buildFunctionURLHeaders(req events.LambdaFunctionURLRequest) http.Header {
	header := make(http.Header)
	for key, v := range req.Headers {
		header.Set(key, v)
	}
	if len(req.Cookies) > 0 {
		header.Set("Cookie", strings.Join(req.Cookies, "; "))
	}
	return header
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
