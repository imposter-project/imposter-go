package websocket

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/emit"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
)

// startSchedules launches connection-scoped schedules declared on the matched
// 'on: open' resource, via the shared emit.ScheduleHost. They stop when the
// connection closes (c.ctx is cancelled) or when their firing limit is reached.
//
// Each firing gets a fresh response state but reuses the connection-scoped
// request store, so values captured by one firing are visible to later ones.
func (c *wsConn) startSchedules(resource *config.BaseResource) {
	host := &emit.ScheduleHost{
		Ctx:            c.ctx,
		Schedules:      resource.Schedule,
		Label:          "connection " + c.upgrade.URL.Path,
		ImposterConfig: c.handler.imposterConfig,
		ConfigDir:      c.handler.config.ConfigDir,
		RespProc:       c.handler.respProc,
		Sink:           frameSink{c},
		NewExchange: func() *exchange.Exchange {
			return exchange.NewExchange(c.upgrade, nil, c.requestStore, response.NewResponseState())
		},
	}
	host.Start(&c.wg)
}
