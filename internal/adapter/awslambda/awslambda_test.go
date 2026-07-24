package awslambda

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertLambdaRequestToHTTPRequest_WithQuery(t *testing.T) {
	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/api/endpoint", "foo=bar&baz=qux", nil, "")
	require.NoError(t, err)

	assert.Equal(t, "/api/endpoint", httpReq.URL.Path)
	assert.Equal(t, "foo=bar&baz=qux", httpReq.URL.RawQuery)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
	assert.Equal(t, "qux", httpReq.URL.Query().Get("baz"))
	assert.Equal(t, "/api/endpoint?foo=bar&baz=qux", httpReq.URL.RequestURI())
}

func TestConvertLambdaRequestToHTTPRequest_WithoutQuery(t *testing.T) {
	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/api/endpoint", "", nil, "")
	require.NoError(t, err)

	assert.Equal(t, "/api/endpoint", httpReq.URL.Path)
	assert.Equal(t, "", httpReq.URL.RawQuery)
	assert.Empty(t, httpReq.URL.Query())
}

func TestConvertLambdaRequestToHTTPRequest_EncodedQueryRoundTrips(t *testing.T) {
	// A raw query with an encoded space and ampersand should decode correctly.
	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/search", "q=hello+world%26more", nil, "")
	require.NoError(t, err)

	assert.Equal(t, "hello world&more", httpReq.URL.Query().Get("q"))
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

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/search", raw, nil, "")
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

	httpReq, err := convertLambdaRequestToHTTPRequest("GET", "/api", raw, nil, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, httpReq.URL.Query()["foo"])
}

func TestBuildAPIGatewayQueryString_Empty(t *testing.T) {
	req := events.APIGatewayProxyRequest{}
	assert.Equal(t, "", buildAPIGatewayQueryString(req))
}

func TestHandleAPIGatewayProxyRequest_PreservesQueryParams(t *testing.T) {
	req := events.APIGatewayProxyRequest{
		HTTPMethod:            "GET",
		Path:                  "/api/endpoint",
		QueryStringParameters: map[string]string{"foo": "bar"},
	}

	httpReq, err := convertLambdaRequestToHTTPRequest(req.HTTPMethod, req.Path, buildAPIGatewayQueryString(req), req.Headers, req.Body)
	require.NoError(t, err)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
}

func TestHandleLambdaFunctionURLRequest_PreservesQueryParams(t *testing.T) {
	req := events.LambdaFunctionURLRequest{
		RawPath:        "/api/endpoint",
		RawQueryString: "foo=bar&baz=qux",
	}

	httpReq, err := convertLambdaRequestToHTTPRequest(req.RequestContext.HTTP.Method, req.RawPath, req.RawQueryString, req.Headers, req.Body)
	require.NoError(t, err)
	assert.Equal(t, "bar", httpReq.URL.Query().Get("foo"))
	assert.Equal(t, "qux", httpReq.URL.Query().Get("baz"))
}
