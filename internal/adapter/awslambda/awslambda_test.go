package awslambda

import (
	"encoding/base64"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{"foo": "bar"},
	}

	raw := buildAPIGatewayQueryString(req)
	assert.Equal(t, "foo=bar", raw)
}

func TestBuildAPIGatewayQueryString_EncodesSpecialCharacters(t *testing.T) {
	// The event delivers already-decoded values; the reconstructed query string
	// must re-encode them so they round-trip through url.Query().
	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{"q": "hello world&more"},
	}

	raw := buildAPIGatewayQueryString(req)

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/search", raw, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello world&more", httpReq.URL.Query().Get("q"))
}

func TestBuildAPIGatewayQueryString_MultiValuePreferred(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{"foo": "b"},
		MultiValueQueryStringParameters: map[string][]string{
			"foo": {"a", "b"},
		},
	}

	raw := buildAPIGatewayQueryString(req)

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/api", raw, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, httpReq.URL.Query()["foo"])
}

func TestBuildAPIGatewayQueryString_Empty(t *testing.T) {
	req := events.APIGatewayProxyRequest{}
	assert.Equal(t, "", buildAPIGatewayQueryString(req))
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
	req := events.APIGatewayProxyRequest{
		Headers: map[string]string{"Content-Type": "application/json"},
	}

	header := buildAPIGatewayHeaders(req)
	assert.Equal(t, "application/json", header.Get("Content-Type"))
}

func TestBuildAPIGatewayHeaders_MultiValuePreferred(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		Headers: map[string]string{"X-Custom": "single"},
		MultiValueHeaders: map[string][]string{
			"X-Custom": {"a", "b"},
		},
	}

	header := buildAPIGatewayHeaders(req)
	assert.Equal(t, []string{"a", "b"}, header.Values("X-Custom"))
}

func TestBuildFunctionURLHeaders_FoldsCookies(t *testing.T) {
	req := events.LambdaFunctionURLRequest{
		Headers: map[string]string{"Content-Type": "text/plain"},
		Cookies: []string{"session=abc", "theme=dark"},
	}

	header := buildFunctionURLHeaders(req)
	assert.Equal(t, "text/plain", header.Get("Content-Type"))
	assert.Equal(t, "session=abc; theme=dark", header.Get("Cookie"))
}

func TestBuildFunctionURLHeaders_NoCookies(t *testing.T) {
	req := events.LambdaFunctionURLRequest{
		Headers: map[string]string{"Accept": "application/json"},
	}

	header := buildFunctionURLHeaders(req)
	assert.Equal(t, "application/json", header.Get("Accept"))
	assert.Empty(t, header.Get("Cookie"))
}

// TestFunctionURLCookiesReadableAsCookie verifies the folded Cookie header is
// parseable by the standard library, so downstream cookie access works.
func TestFunctionURLCookiesReadableAsCookie(t *testing.T) {
	req := events.LambdaFunctionURLRequest{
		RawPath: "/api",
		Cookies: []string{"session=abc", "theme=dark"},
	}

	httpReq, err := convertLambdaRequestToHTTPRequest(req.RequestContext.HTTP.Method, req.RawPath, req.RawQueryString, buildFunctionURLHeaders(req), nil)
	require.NoError(t, err)

	c, err := httpReq.Cookie("session")
	require.NoError(t, err)
	assert.Equal(t, "abc", c.Value)
	c, err = httpReq.Cookie("theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", c.Value)
}

func TestHandleAPIGatewayProxyRequest_PreservesQueryParams(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		HTTPMethod:            "GET",
		Path:                  "/api/endpoint",
		QueryStringParameters: map[string]string{"foo": "bar"},
	}

	httpReq, err := convertLambdaRequestToHTTPRequest(req.HTTPMethod, req.Path, buildAPIGatewayQueryString(req), buildAPIGatewayHeaders(req), []byte(req.Body))
	require.NoError(t, err)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
}

// TestFormURLEncodedBodyParses exercises the full path a form POST takes: the
// body and Content-Type header must survive conversion so ParseForm works.
func TestFormURLEncodedBodyParses(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/submit",
		Headers:    map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:       "name=alice&role=admin",
	}

	body, err := decodeLambdaBody(req.Body, req.IsBase64Encoded)
	require.NoError(t, err)
	httpReq, err := convertLambdaRequestToHTTPRequest(req.HTTPMethod, req.Path, buildAPIGatewayQueryString(req), buildAPIGatewayHeaders(req), body)
	require.NoError(t, err)

	require.NoError(t, httpReq.ParseForm())
	assert.Equal(t, "alice", httpReq.PostFormValue("name"))
	assert.Equal(t, "admin", httpReq.PostFormValue("role"))
}

// TestBase64FormBodyParses verifies a base64-encoded form body (as API Gateway
// delivers binary/opaque payloads) is decoded before form parsing.
func TestBase64FormBodyParses(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		HTTPMethod:      "POST",
		Path:            "/submit",
		Headers:         map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:            base64.StdEncoding.EncodeToString([]byte("name=bob&role=user")),
		IsBase64Encoded: true,
	}

	body, err := decodeLambdaBody(req.Body, req.IsBase64Encoded)
	require.NoError(t, err)
	httpReq, err := convertLambdaRequestToHTTPRequest(req.HTTPMethod, req.Path, buildAPIGatewayQueryString(req), buildAPIGatewayHeaders(req), body)
	require.NoError(t, err)

	require.NoError(t, httpReq.ParseForm())
	assert.Equal(t, "bob", httpReq.PostFormValue("name"))
	assert.Equal(t, "user", httpReq.PostFormValue("role"))
}

func TestHandleLambdaFunctionURLRequest_PreservesQueryParams(t *testing.T) {
	req := events.LambdaFunctionURLRequest{
		RawPath:        "/api/endpoint",
		RawQueryString: "foo=bar&baz=qux",
	}

	httpReq, err := convertLambdaRequestToHTTPRequest(req.RequestContext.HTTP.Method, req.RawPath, req.RawQueryString, buildFunctionURLHeaders(req), []byte(req.Body))
	require.NoError(t, err)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
	assert.Equal(t, "qux", httpReq.URL.Query().Get("baz"))
}
