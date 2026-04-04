package rest

import (
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/pipeline"
	"github.com/imposter-project/imposter-go/internal/response"
)

// HandleRequest processes incoming REST API requests
func (h *PluginHandler) HandleRequest(
	exch *exchange.Exchange,
	respProc response.Processor,
) {
	pipeline.RunPipeline(h.config, h.imposterConfig, exch, respProc, nil)
}
