package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

// newRateLimitExchange builds an exchange with a fresh response state for a
// rate limiting test.
func newRateLimitExchange() *exchange.Exchange {
	req := httptest.NewRequest("GET", "/limited", nil)
	return exchange.NewExchange(req, nil, store.NewRequestStore(), response.NewResponseState())
}

// newLimitedResource builds a resource whose concurrency limit is exceeded by
// the first request, since limits match when the count is greater than the
// threshold.
func newLimitedResource(resourceID string, limitResponse *config.Response) *config.Resource {
	return &config.Resource{
		BaseResource: config.BaseResource{
			RequestMatcher: config.RequestMatcher{Method: "GET", Path: "/limited"},
			Concurrency: []config.ConcurrencyLimit{
				{Threshold: 0, Response: limitResponse},
			},
			ResourceID: resourceID,
		},
	}
}

// passThroughProcessor applies the response configuration using the standard
// processor, mirroring how the pipeline invokes rate limiting.
func passThroughProcessor(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	resp *config.Response,
	respProc response.Processor,
) {
	respProc(exch, reqMatcher, resp)
}

// TestRateLimitCheck_NoConcurrencyConfig checks that a resource without any
// concurrency limits is never rate limited.
func TestRateLimitCheck_NoConcurrencyConfig(t *testing.T) {
	store.InitStoreProvider()
	exch := newRateLimitExchange()

	resource := &config.Resource{
		BaseResource: config.BaseResource{
			RequestMatcher: config.RequestMatcher{Method: "GET", Path: "/unlimited"},
			ResourceID:     "no-concurrency",
		},
	}

	limited := RateLimitCheck(resource, "GET", "/unlimited", exch, response.NewProcessor(&config.ImposterConfig{}, t.TempDir()), passThroughProcessor)

	assert.False(t, limited, "resource without concurrency limits should not be rate limited")
	assert.Empty(t, exch.ResponseState.CleanupFunctions, "no cleanup should be registered when there are no limits")
}

// TestRateLimitCheck_UnderThreshold checks that a request below the threshold
// proceeds and registers a cleanup function to release its slot.
func TestRateLimitCheck_UnderThreshold(t *testing.T) {
	store.InitStoreProvider()
	exch := newRateLimitExchange()

	resource := &config.Resource{
		BaseResource: config.BaseResource{
			RequestMatcher: config.RequestMatcher{Method: "GET", Path: "/limited"},
			Concurrency: []config.ConcurrencyLimit{
				{Threshold: 10, Response: &config.Response{StatusCode: http.StatusTooManyRequests}},
			},
			ResourceID: "under-threshold",
		},
	}

	limited := RateLimitCheck(resource, "GET", "/limited", exch, response.NewProcessor(&config.ImposterConfig{}, t.TempDir()), passThroughProcessor)

	assert.False(t, limited, "request below the threshold should not be rate limited")
	assert.False(t, exch.ResponseState.Handled, "response should not be handled when under the threshold")
	assert.Len(t, exch.ResponseState.CleanupFunctions, 1, "a cleanup function should release the slot")

	// release the slot so the global counter does not leak into other tests
	exch.ResponseState.CleanupFunctions[0]()
}

// TestRateLimitCheck_OverThreshold checks that an exceeded limit applies the
// configured limit response and marks the request as handled.
func TestRateLimitCheck_OverThreshold(t *testing.T) {
	store.InitStoreProvider()
	exch := newRateLimitExchange()

	resource := newLimitedResource("over-threshold", &config.Response{
		StatusCode: http.StatusTooManyRequests,
		Content:    "slow down",
	})

	limited := RateLimitCheck(resource, "GET", "/limited", exch, response.NewProcessor(&config.ImposterConfig{}, t.TempDir()), passThroughProcessor)

	assert.True(t, limited, "request over the threshold should be rate limited")
	assert.Equal(t, http.StatusTooManyRequests, exch.ResponseState.StatusCode)
	assert.Equal(t, "slow down", string(exch.ResponseState.Body))
	assert.True(t, exch.ResponseState.Handled, "rate limited response should be marked as handled")
	assert.Equal(t, &resource.BaseResource, exch.ResponseState.Resource)
}

// TestRateLimitCheck_OverridesScriptSetResponse checks that the limit response
// replaces values an earlier interceptor's script set explicitly. Script-set
// values otherwise take precedence over configured ones, so the rate limiter
// must clear that marking for its own response to apply.
func TestRateLimitCheck_OverridesScriptSetResponse(t *testing.T) {
	store.InitStoreProvider()
	exch := newRateLimitExchange()

	// simulate an interceptor script having set the response properties
	exch.ResponseState.SetStatusCode(http.StatusOK)
	exch.ResponseState.SetBody([]byte("from-script"))
	exch.ResponseState.SetHeader("X-Source", "script")

	resource := newLimitedResource("overrides-script", &config.Response{
		StatusCode: http.StatusTooManyRequests,
		Content:    "slow down",
		Headers:    map[string]string{"X-Source": "rate-limit"},
	})

	limited := RateLimitCheck(resource, "GET", "/limited", exch, response.NewProcessor(&config.ImposterConfig{}, t.TempDir()), passThroughProcessor)

	assert.True(t, limited, "request over the threshold should be rate limited")
	assert.Equal(t, http.StatusTooManyRequests, exch.ResponseState.StatusCode,
		"limit response status code should override the script-set status code")
	assert.Equal(t, "slow down", string(exch.ResponseState.Body),
		"limit response content should override the script-set body")
	assert.Equal(t, "rate-limit", exch.ResponseState.Headers["X-Source"],
		"limit response headers should override script-set headers")
}

// TestRateLimitCheck_GeneratesKeyWhenResourceIDMissing checks that rate
// limiting still applies when the resource ID has not been pre-calculated.
func TestRateLimitCheck_GeneratesKeyWhenResourceIDMissing(t *testing.T) {
	store.InitStoreProvider()
	exch := newRateLimitExchange()

	resource := newLimitedResource("", &config.Response{StatusCode: http.StatusTooManyRequests})

	limited := RateLimitCheck(resource, "GET", "/limited", exch, response.NewProcessor(&config.ImposterConfig{}, t.TempDir()), passThroughProcessor)

	assert.True(t, limited, "rate limiting should apply using a runtime-generated key")
	assert.Equal(t, http.StatusTooManyRequests, exch.ResponseState.StatusCode)
}
