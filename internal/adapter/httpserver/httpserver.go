package httpserver

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gatehill/imposter-go/internal/config"
	"github.com/gatehill/imposter-go/internal/handler"
	"github.com/gatehill/imposter-go/internal/store"
)

// httpServer represents the HTTP server configuration.
type httpServer struct {
	Addr      string
	ConfigDir string
	Configs   []config.Config
}

// StartServer initializes and starts the HTTP server.
func StartServer() {
	fmt.Println("Starting Imposter-Go...")

	imposterConfig := config.LoadImposterConfig()

	if len(os.Args) < 2 {
		panic("Config directory path must be provided as the first argument")
	}

	configDir := os.Args[1]
	if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
		panic("Specified path is not a valid directory")
	}

	configs := config.LoadConfig(configDir)

	store.InitStoreProvider()
	store.PreloadStores(configDir, configs)

	// Initialize and start the server with multiple configs
	srv := newServer(imposterConfig, configDir, configs)
	srv.start(imposterConfig)
}

// newServer creates a new instance of httpServer.
func newServer(imposterConfig *config.ImposterConfig, configDir string, configs []config.Config) *httpServer {
	return &httpServer{
		Addr:      ":" + imposterConfig.ServerPort,
		ConfigDir: configDir,
		Configs:   configs,
	}
}

// start begins listening for HTTP requests and handles them.
func (s *httpServer) start(imposterConfig *config.ImposterConfig) {
	fmt.Printf("Server is listening on %s...\n", s.Addr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(w, r, s.ConfigDir, s.Configs, imposterConfig)
	})

	http.HandleFunc("/system/store/", handler.HandleStoreRequest)

	if err := http.ListenAndServe(s.Addr, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
