package httpserver

import (
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/imposter-project/imposter-go/internal/adapter"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/pkg/logger"
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
// When TLS is configured, the server uses h2 (HTTP/2 over TLS).
// Otherwise it uses h2c (HTTP/2 cleartext) for backwards compatibility.
func (s *httpServer) start(imposterConfig *config.ImposterConfig) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(imposterConfig, w, r, s.Plugins)
	})

	if imposterConfig.TLSEnabled() {
		logger.Infof("server is listening on %s (h2/TLS)...", s.Addr)
		server := &http.Server{
			Addr:    s.Addr,
			Handler: mux,
		}
		if err := server.ListenAndServeTLS(imposterConfig.TLSCertFile, imposterConfig.TLSKeyFile); err != nil {
			logger.Errorf("error starting TLS server: %v", err)
		}
	} else {
		logger.Infof("server is listening on %s (h2c)...", s.Addr)
		h2s := &http2.Server{}
		h2cHandler := h2c.NewHandler(mux, h2s)
		server := &http.Server{
			Addr:    s.Addr,
			Handler: h2cHandler,
		}
		if err := server.ListenAndServe(); err != nil {
			logger.Errorf("error starting server: %v", err)
		}
	}
}
