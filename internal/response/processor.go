package response

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
)

// Processor is a function that plugins implement to handle their specific response processing
type Processor func(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	resp *config.Response,
)

// NewProcessor creates a new standard response processor
func NewProcessor(imposterConfig *config.ImposterConfig, configDir string) Processor {
	return func(exch *exchange.Exchange, reqMatcher *config.RequestMatcher, resp *config.Response) {
		processResponse(exch, reqMatcher, resp, configDir, imposterConfig)
	}
}
