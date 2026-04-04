package httpserver

import (
	"crypto/tls"
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
// When TLS is configured, the server uses h2 (HTTP/2 over TLS), or HTTPS/1.1
// when HTTP/2 is disabled. Otherwise it uses h2c (HTTP/2 cleartext), or plain
// HTTP/1.1 when HTTP/2 is disabled.
func (s *httpServer) start(imposterConfig *config.ImposterConfig) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(imposterConfig, w, r, s.Plugins)
	})

	if imposterConfig.TLSEnabled() {
		server := &http.Server{
			Addr:    s.Addr,
			Handler: mux,
		}
		if imposterConfig.HTTP2Enabled {
			logger.Infof("server is listening on %s (h2/TLS)...", s.Addr)
		} else {
			// Suppress ALPN upgrade to h2, forcing HTTPS/1.1.
			server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
			logger.Infof("server is listening on %s (https/1.1)...", s.Addr)
		}
		if err := server.ListenAndServeTLS(imposterConfig.TLSCertFile, imposterConfig.TLSKeyFile); err != nil {
			logger.Errorf("error starting TLS server: %v", err)
		}
	} else {
		var httpHandler http.Handler = mux
		if imposterConfig.HTTP2Enabled {
			h2s := &http2.Server{}
			httpHandler = h2c.NewHandler(mux, h2s)
			logger.Infof("server is listening on %s (h2c)...", s.Addr)
		} else {
			logger.Infof("server is listening on %s (http/1.1)...", s.Addr)
		}
		server := &http.Server{
			Addr:    s.Addr,
			Handler: httpHandler,
		}
		if err := server.ListenAndServe(); err != nil {
			logger.Errorf("error starting server: %v", err)
		}
	}
}
