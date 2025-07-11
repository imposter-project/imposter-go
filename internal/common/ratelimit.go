package common

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/ratelimiter"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// RateLimitCheck performs rate limiting check and returns true if request should be rate limited
func RateLimitCheck(
	resource *config.Resource,
	resourceMethod string,
	resourceName string,
	exch *exchange.Exchange,
	respProc response.Processor,
	processResponseFunc func(*exchange.Exchange, *config.RequestMatcher, *config.Response, response.Processor),
) (shouldLimit bool) {
	if len(resource.Concurrency) == 0 {
		return false
	}

	rateLimiter := ratelimiter.GetGlobalRateLimiter()

	resourceKey := resource.ResourceID
	if resourceKey == "" {
		// Fallback: generate key if not pre-calculated (dynamically created keys)
		logger.Warnf("resource ID not pre-calculated for resource, generating at runtime")
		resourceKey = config.GenerateResourceKey(resourceMethod, resourceName, &resource.RequestMatcher)
	}

	if limitResponse, err := rateLimiter.CheckAndIncrement(resourceKey, resource.Concurrency); limitResponse != nil {
		// Rate limit exceeded, return the configured response
		if err != nil {
			logger.Warnf("rate limiter error: %v", err)
		}
		logger.Infof("rate limit applied for resource %s", resourceKey)
		processResponseFunc(exch, &resource.RequestMatcher, limitResponse.Response, respProc)
		exch.ResponseState.HandledWithResource(&resource.BaseResource)
		return true
	}

	// Register cleanup function with ResponseState
	cleanupFunc := func() {
		if err := rateLimiter.Decrement(resourceKey); err != nil {
			logger.Warnf("failed to decrement rate limiter count: %v", err)
		}
	}
	exch.ResponseState.CleanupFunctions = append(exch.ResponseState.CleanupFunctions, cleanupFunc)

	return false
}
