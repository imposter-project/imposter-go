package awslambda

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLambdaHTTPEvent_DecodesAPIGatewayV1 verifies a v1 payload maps onto the
// unified event with a single decode, and is distinguishable as v1.
func TestLambdaHTTPEvent_DecodesAPIGatewayV1(t *testing.T) {
	payload := []byte(`{
		"httpMethod":"GET",
		"path":"/api",
		"headers":{"Content-Type":"application/json"},
		"multiValueHeaders":{"X-Custom":["a","b"]},
		"queryStringParameters":{"foo":"bar"},
		"multiValueQueryStringParameters":{"foo":["a","b"]},
		"body":"hello",
		"isBase64Encoded":false
	}`)

	var evt lambdaHTTPEvent
	require.NoError(t, json.Unmarshal(payload, &evt))

	assert.Equal(t, "GET", evt.HTTPMethod)
	assert.Empty(t, evt.RequestContext.HTTP.Method) // not a v2 event
	assert.Equal(t, "/api", evt.Path)
	assert.Equal(t, "bar", evt.QueryStringParameters["foo"])
	assert.Equal(t, []string{"a", "b"}, evt.MultiValueHeaders["X-Custom"])
	assert.Equal(t, "hello", evt.Body)
}

// TestLambdaHTTPEvent_DecodesFunctionURLV2 verifies a v2 payload maps onto the
// unified event with a single decode, and is distinguishable as v2.
func TestLambdaHTTPEvent_DecodesFunctionURLV2(t *testing.T) {
	payload := []byte(`{
		"version":"2.0",
		"rawPath":"/api",
		"rawQueryString":"foo=bar",
		"cookies":["session=abc","theme=dark"],
		"headers":{"content-type":"text/plain"},
		"requestContext":{"http":{"method":"POST"}},
		"body":"hi"
	}`)

	var evt lambdaHTTPEvent
	require.NoError(t, json.Unmarshal(payload, &evt))

	assert.Empty(t, evt.HTTPMethod) // not a v1 event
	assert.Equal(t, "POST", evt.RequestContext.HTTP.Method)
	assert.Equal(t, "/api", evt.RawPath)
	assert.Equal(t, "foo=bar", evt.RawQueryString)
	assert.Equal(t, []string{"session=abc", "theme=dark"}, evt.Cookies)
}

func TestConvertLambdaRequestToHTTPRequest_WithQuery(t *testing.T) {
	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/api/endpoint", "foo=bar&baz=qux", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "/api/endpoint", httpReq.URL.Path)
	assert.Equal(t, "foo=bar&baz=qux", httpReq.URL.RawQuery)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
	assert.Equal(t, "qux", httpReq.URL.Query().Get("baz"))
	assert.Equal(t, "/api/endpoint?foo=bar&baz=qux", httpReq.URL.RequestURI())
}

func TestConvertLambdaRequestToHTTPRequest_WithoutQuery(t *testing.T) {
	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/api/endpoint", "", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "/api/endpoint", httpReq.URL.Path)
	assert.Equal(t, "", httpReq.URL.RawQuery)
	assert.Empty(t, httpReq.URL.Query())
}

func TestConvertLambdaRequestToHTTPRequest_EncodedQueryRoundTrips(t *testing.T) {
	// A raw query with an encoded space and ampersand should decode correctly.
	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/search", "q=hello+world%26more", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "hello world&more", httpReq.URL.Query().Get("q"))
}

func TestConvertLambdaRequestToHTTPRequest_BodyAndHeaders(t *testing.T) {
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	httpReq, err := convertLambdaRequestToHTTPRequest("POST", "/api", "", header, []byte(`{"key":"value"}`))
	require.NoError(t, err)

	assert.Equal(t, "application/json", httpReq.Header.Get("Content-Type"))
	got, err := io.ReadAll(httpReq.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"key":"value"}`, string(got))
}

func TestBuildAPIGatewayQueryString_SingleValue(t *testing.T) {
	evt := lambdaHTTPEvent{
		QueryStringParameters: map[string]string{"foo": "bar"},
	}

	raw := buildAPIGatewayQueryString(evt)
	assert.Equal(t, "foo=bar", raw)
}

func TestBuildAPIGatewayQueryString_EncodesSpecialCharacters(t *testing.T) {
	// The event delivers already-decoded values; the reconstructed query string
	// must re-encode them so they round-trip through url.Query().
	evt := lambdaHTTPEvent{
		QueryStringParameters: map[string]string{"q": "hello world&more"},
	}

	raw := buildAPIGatewayQueryString(evt)

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/search", raw, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello world&more", httpReq.URL.Query().Get("q"))
}

func TestBuildAPIGatewayQueryString_MultiValuePreferred(t *testing.T) {
	evt := lambdaHTTPEvent{
		QueryStringParameters: map[string]string{"foo": "b"},
		MultiValueQueryStringParameters: map[string][]string{
			"foo": {"a", "b"},
		},
	}

	raw := buildAPIGatewayQueryString(evt)

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/api", raw, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, httpReq.URL.Query()["foo"])
}

func TestBuildAPIGatewayQueryString_Empty(t *testing.T) {
	assert.Equal(t, "", buildAPIGatewayQueryString(lambdaHTTPEvent{}))
}

func TestDecodeLambdaBody_PlainText(t *testing.T) {
	body, err := decodeLambdaBody("hello world", false)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello world"), body)
}

func TestDecodeLambdaBody_Base64Binary(t *testing.T) {
	binary := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	encoded := base64.StdEncoding.EncodeToString(binary)

	body, err := decodeLambdaBody(encoded, true)
	require.NoError(t, err)
	assert.Equal(t, binary, body)
}

func TestDecodeLambdaBody_InvalidBase64(t *testing.T) {
	_, err := decodeLambdaBody("not-valid-base64!!!", true)
	assert.Error(t, err)
}

func TestBuildAPIGatewayHeaders_SingleValue(t *testing.T) {
	evt := lambdaHTTPEvent{
		Headers: map[string]string{"Content-Type": "application/json"},
	}

	header := buildAPIGatewayHeaders(evt)
	assert.Equal(t, "application/json", header.Get("Content-Type"))
}

func TestBuildAPIGatewayHeaders_MultiValuePreferred(t *testing.T) {
	evt := lambdaHTTPEvent{
		Headers: map[string]string{"X-Custom": "single"},
		MultiValueHeaders: map[string][]string{
			"X-Custom": {"a", "b"},
		},
	}

	header := buildAPIGatewayHeaders(evt)
	assert.Equal(t, []string{"a", "b"}, header.Values("X-Custom"))
}

func TestBuildFunctionURLHeaders_FoldsCookies(t *testing.T) {
	evt := lambdaHTTPEvent{
		Headers: map[string]string{"Content-Type": "text/plain"},
		Cookies: []string{"session=abc", "theme=dark"},
	}

	header := buildFunctionURLHeaders(evt)
	assert.Equal(t, "text/plain", header.Get("Content-Type"))
	assert.Equal(t, "session=abc; theme=dark", header.Get("Cookie"))
}

func TestBuildFunctionURLHeaders_NoCookies(t *testing.T) {
	evt := lambdaHTTPEvent{
		Headers: map[string]string{"Accept": "application/json"},
	}

	header := buildFunctionURLHeaders(evt)
	assert.Equal(t, "application/json", header.Get("Accept"))
	assert.Empty(t, header.Get("Cookie"))
}

// TestFunctionURLCookiesReadableAsCookie verifies the folded Cookie header is
// parseable by the standard library, so downstream cookie access works.
func TestFunctionURLCookiesReadableAsCookie(t *testing.T) {
	evt := lambdaHTTPEvent{
		RawPath: "/api",
		Cookies: []string{"session=abc", "theme=dark"},
	}

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", evt.RawPath, evt.RawQueryString, buildFunctionURLHeaders(evt), nil)
	require.NoError(t, err)

	c, err := httpReq.Cookie("session")
	require.NoError(t, err)
	assert.Equal(t, "abc", c.Value)
	c, err = httpReq.Cookie("theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", c.Value)
}

// TestAPIGatewayQueryParamsReachRequest exercises the v1 assembly: query params
// from the event reach the http.Request.
func TestAPIGatewayQueryParamsReachRequest(t *testing.T) {
	evt := lambdaHTTPEvent{
		HTTPMethod:            "GET",
		Path:                  "/api/endpoint",
		QueryStringParameters: map[string]string{"foo": "bar"},
	}

	httpReq, err := convertLambdaRequestToHTTPRequest(evt.HTTPMethod, evt.Path, buildAPIGatewayQueryString(evt), buildAPIGatewayHeaders(evt), []byte(evt.Body))
	require.NoError(t, err)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
}

// TestFormURLEncodedBodyParses exercises the full path a form POST takes: the
// body and Content-Type header must survive conversion so ParseForm works.
func TestFormURLEncodedBodyParses(t *testing.T) {
	evt := lambdaHTTPEvent{
		HTTPMethod: "POST",
		Path:       "/submit",
		Headers:    map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:       "name=alice&role=admin",
	}

	body, err := decodeLambdaBody(evt.Body, evt.IsBase64Encoded)
	require.NoError(t, err)
	httpReq, err := convertLambdaRequestToHTTPRequest(evt.HTTPMethod, evt.Path, buildAPIGatewayQueryString(evt), buildAPIGatewayHeaders(evt), body)
	require.NoError(t, err)

	require.NoError(t, httpReq.ParseForm())
	assert.Equal(t, "alice", httpReq.PostFormValue("name"))
	assert.Equal(t, "admin", httpReq.PostFormValue("role"))
}

// TestBase64FormBodyParses verifies a base64-encoded form body (as API Gateway
// delivers binary/opaque payloads) is decoded before form parsing.
func TestBase64FormBodyParses(t *testing.T) {
	evt := lambdaHTTPEvent{
		HTTPMethod:      "POST",
		Path:            "/submit",
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:            base64.StdEncoding.EncodeToString([]byte("name=bob&role=user")),
		IsBase64Encoded: true,
	}

	body, err := decodeLambdaBody(evt.Body, evt.IsBase64Encoded)
	require.NoError(t, err)
	httpReq, err := convertLambdaRequestToHTTPRequest(evt.HTTPMethod, evt.Path, buildAPIGatewayQueryString(evt), buildAPIGatewayHeaders(evt), body)
	require.NoError(t, err)

	require.NoError(t, httpReq.ParseForm())
	assert.Equal(t, "bob", httpReq.PostFormValue("name"))
	assert.Equal(t, "user", httpReq.PostFormValue("role"))
}

// TestFunctionURLQueryParamsReachRequest exercises the v2 assembly: the raw
// query string reaches the http.Request.
func TestFunctionURLQueryParamsReachRequest(t *testing.T) {
	evt := lambdaHTTPEvent{
		RawPath:        "/api/endpoint",
		RawQueryString: "foo=bar&baz=qux",
	}

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", evt.RawPath, evt.RawQueryString, buildFunctionURLHeaders(evt), []byte(evt.Body))
	require.NoError(t, err)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
	assert.Equal(t, "qux", httpReq.URL.Query().Get("baz"))
}
