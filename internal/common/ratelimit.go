package common

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/ratelimiter"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/internal/system"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// RateLimitCheck performs rate limiting check and returns true if request should be rate limited
// defaultMethod is used when resource.Method is empty (e.g., "POST" for SOAP)
// resourcePath is used instead of resource.Path for plugins that need custom path logic (e.g., SOAP using operation name)
func RateLimitCheck(
	resource *config.Resource,
	resourceMethod string,
	resourcePath string,
	exch *exchange.Exchange,
	respProc response.Processor,
	processResponseFunc func(*exchange.Exchange, *config.RequestMatcher, *config.Response, response.Processor),
) (shouldLimit bool, cleanupFunc func()) {
	if len(resource.Concurrency) == 0 {
		return false, nil
	}

	resourceKey := ratelimiter.GenerateResourceKey(resourceMethod, resourcePath)

	storeProvider := store.GetStoreProvider()
	instanceID := system.GenerateInstanceID()
	rateLimiter := ratelimiter.NewRateLimiter(storeProvider)

	if limitResponse, err := rateLimiter.CheckAndIncrement(resourceKey, resource.Concurrency, instanceID); limitResponse != nil {
		// Rate limit exceeded, return the configured response
		if err != nil {
			logger.Warnf("rate limiter error: %v", err)
		}
		logger.Infof("rate limit applied for resource %s", resourceKey)
		processResponseFunc(exch, &resource.RequestMatcher, limitResponse.Response, respProc)
		exch.ResponseState.HandledWithResource(&resource.BaseResource)
		return true, nil
	}

	// Set up cleanup function
	cleanupFunc = func() {
		if err := rateLimiter.Decrement(resourceKey, instanceID); err != nil {
			logger.Warnf("failed to decrement rate limiter count: %v", err)
		}
	}

	return false, cleanupFunc
}
