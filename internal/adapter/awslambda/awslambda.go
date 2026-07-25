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

// lambdaHTTPEvent holds the fields of the supported Lambda HTTP event shapes
// that the adapter consumes, so an incoming payload is JSON-decoded exactly
// once regardless of its shape. API Gateway proxy (v1) carries a top-level
// httpMethod; Function URL / API Gateway HTTP API (v2) carries
// requestContext.http.method instead. AWS routing metadata not read by the
// handlers (e.g. stageVariables, pathParameters, resource) is intentionally
// omitted.
type lambdaHTTPEvent struct {
	// API Gateway proxy (v1)
	HTTPMethod                      string              `json:"httpMethod"`
	Path                            string              `json:"path"`
	MultiValueHeaders               map[string][]string `json:"multiValueHeaders"`
	QueryStringParameters           map[string]string   `json:"queryStringParameters"`
	MultiValueQueryStringParameters map[string][]string `json:"multiValueQueryStringParameters"`

	// Function URL / API Gateway HTTP API (v2)
	RawPath        string   `json:"rawPath"`
	RawQueryString string   `json:"rawQueryString"`
	Cookies        []string `json:"cookies"`
	RequestContext struct {
		HTTP struct {
			Method string `json:"method"`
		} `json:"http"`
	} `json:"requestContext"`

	// Shared
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	IsBase64Encoded bool              `json:"isBase64Encoded"`
}

// HandleLambdaRequest handles incoming Lambda requests and routes them to the
// appropriate handler. The payload is decoded once into a lambdaHTTPEvent and
// the event shape is determined from its discriminating fields, so neither
// shape is ever decoded twice.
func HandleLambdaRequest(req json.RawMessage) (interface{}, error) {
	var evt lambdaHTTPEvent
	if err := json.Unmarshal(req, &evt); err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: 400, Body: "Unsupported request type"}, nil
	}

	switch {
	case evt.HTTPMethod != "":
		return handleAPIGatewayProxyRequest(evt, plugins)
	case evt.RequestContext.HTTP.Method != "":
		return handleLambdaFunctionURLRequest(evt, plugins)
	default:
		return events.LambdaFunctionURLResponse{StatusCode: 400, Body: "Unsupported request type"}, nil
	}
}

// handleAPIGatewayProxyRequest processes API Gateway proxy (v1) requests.
func handleAPIGatewayProxyRequest(evt lambdaHTTPEvent, plugins []plugin.Plugin) (events.APIGatewayProxyResponse, error) {
	// Decode the body first: API Gateway base64-encodes binary payloads (e.g.
	// file uploads and multipart bodies) and sets IsBase64Encoded accordingly.
	body, err := decodeLambdaBody(evt.Body, evt.IsBase64Encoded)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "Failed to decode request body"}, nil
	}

	// Preserve query parameters and headers so that route matching and script
	// access (queryParams, headers, formParams derived from the body) work.
	httpReq, err := convertLambdaRequestToHTTPRequest(evt.HTTPMethod, evt.Path, buildAPIGatewayQueryString(evt), buildAPIGatewayHeaders(evt), body)
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

// handleLambdaFunctionURLRequest processes Function URL / API Gateway HTTP API
// (v2) requests.
func handleLambdaFunctionURLRequest(evt lambdaHTTPEvent, plugins []plugin.Plugin) (events.LambdaFunctionURLResponse, error) {
	// Decode the body first: the v2 payload base64-encodes binary payloads (e.g.
	// file uploads and multipart bodies) and sets IsBase64Encoded accordingly.
	body, err := decodeLambdaBody(evt.Body, evt.IsBase64Encoded)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: 400, Body: "Failed to decode request body"}, nil
	}

	// RawQueryString is already URL-encoded in the v2 payload, so it is used
	// directly. Cookies arrive in a separate array (not the headers map) and are
	// folded back into a Cookie header by buildFunctionURLHeaders.
	httpReq, err := convertLambdaRequestToHTTPRequest(evt.RequestContext.HTTP.Method, evt.RawPath, evt.RawQueryString, buildFunctionURLHeaders(evt), body)
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
// Gateway proxy (v1) event. Parameter values in the event are already
// URL-decoded, so they are re-encoded here. MultiValueQueryStringParameters is
// preferred when present because it preserves repeated keys (e.g. ?foo=a&foo=b);
// otherwise the single-valued QueryStringParameters map is used.
func buildAPIGatewayQueryString(evt lambdaHTTPEvent) string {
	values := url.Values{}
	if len(evt.MultiValueQueryStringParameters) > 0 {
		for key, vals := range evt.MultiValueQueryStringParameters {
			for _, v := range vals {
				values.Add(key, v)
			}
		}
	} else {
		for key, v := range evt.QueryStringParameters {
			values.Set(key, v)
		}
	}
	return values.Encode()
}

// buildAPIGatewayHeaders converts the request headers of an API Gateway proxy
// (v1) event to http.Header. MultiValueHeaders is preferred when present so
// repeated headers (e.g. multiple Cookie or Accept values) are preserved;
// otherwise the single-valued Headers map is used.
func buildAPIGatewayHeaders(evt lambdaHTTPEvent) http.Header {
	header := make(http.Header)
	if len(evt.MultiValueHeaders) > 0 {
		for key, vals := range evt.MultiValueHeaders {
			for _, v := range vals {
				header.Add(key, v)
			}
		}
	} else {
		for key, v := range evt.Headers {
			header.Set(key, v)
		}
	}
	return header
}

// buildFunctionURLHeaders converts the request headers of a Lambda Function URL
// / API Gateway HTTP API (v2) event to http.Header. In the v2 payload format
// cookies are delivered in a separate Cookies array rather than the headers
// map, so they are recombined into a single Cookie header (RFC 6265).
func buildFunctionURLHeaders(evt lambdaHTTPEvent) http.Header {
	header := make(http.Header)
	for key, v := range evt.Headers {
		header.Set(key, v)
	}
	if len(evt.Cookies) > 0 {
		header.Set("Cookie", strings.Join(evt.Cookies, "; "))
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
