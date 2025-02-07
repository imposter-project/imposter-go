package httpserver

import (
	"github.com/imposter-project/imposter-go/pkg/logger"
	"net/http"
	"os"
	"time"

	"github.com/imposter-project/imposter-go/internal/adapter"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/plugin"
)

// HTTPAdapter represents the HTTP server runtime adapter
type HTTPAdapter struct{}

// NewAdapter creates a new HTTP server adapter instance
func NewAdapter() adapter.Adapter {
	return &HTTPAdapter{}
}

// Start begins the HTTP server runtime
func (a *HTTPAdapter) Start() {
	startTime := time.Now()
	var configDirArg string
	if len(os.Args) >= 2 {
		configDirArg = os.Args[1]
	}

	imposterConfig, configs := adapter.InitialiseImposter(configDirArg)

	// Initialise and start the server with multiple configs
	srv := newServer(imposterConfig, configs)
	logger.Infof("startup completed in %v", time.Since(startTime))
	srv.start(imposterConfig)
}

// httpServer represents the HTTP server configuration.
type httpServer struct {
	Addr    string
	Plugins []plugin.Plugin
}

// newServer creates a new instance of httpServer.
func newServer(imposterConfig *config.ImposterConfig, plugins []plugin.Plugin) *httpServer {
	return &httpServer{
		Addr:    ":" + imposterConfig.ServerPort,
		Plugins: plugins,
	}
}

// start begins listening for HTTP requests and handles them.
func (s *httpServer) start(imposterConfig *config.ImposterConfig) {
	logger.Infof("server is listening on %s...", s.Addr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(imposterConfig, w, r, s.Plugins)
	})

	if err := http.ListenAndServe(s.Addr, nil); err != nil {
		logger.Errorf("error starting server: %v", err)
	}
}
