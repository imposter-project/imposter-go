package httpserver

import (
	"net/http"
	"os"
	"time"

	"github.com/imposter-project/imposter-go/internal/adapter"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/internal/logger"
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

	imposterConfig, configDir, configs := adapter.InitialiseImposter(configDirArg)

	// Initialise and start the server with multiple configs
	srv := newServer(imposterConfig, configDir, configs)
	logger.Infof("startup completed in %v", time.Since(startTime))
	srv.start(imposterConfig)
}

// httpServer represents the HTTP server configuration.
type httpServer struct {
	Addr      string
	ConfigDir string
	Plugins   []plugin.Plugin
}

// newServer creates a new instance of httpServer.
func newServer(imposterConfig *config.ImposterConfig, configDir string, plugins []plugin.Plugin) *httpServer {
	return &httpServer{
		Addr:      ":" + imposterConfig.ServerPort,
		ConfigDir: configDir,
		Plugins:   plugins,
	}
}

// start begins listening for HTTP requests and handles them.
func (s *httpServer) start(imposterConfig *config.ImposterConfig) {
	logger.Infof("server is listening on %s...", s.Addr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(w, r, s.ConfigDir, s.Plugins, imposterConfig)
	})

	if err := http.ListenAndServe(s.Addr, nil); err != nil {
		logger.Errorf("error starting server: %v", err)
	}
}
