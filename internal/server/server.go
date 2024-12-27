package server

import (
	"fmt"
	"github.com/gatehill/imposter-go/internal/config"
	"github.com/gatehill/imposter-go/internal/handler"
	"net/http"
)

type Server struct {
	Addr      string
	ConfigDir string
	Configs   []config.Config
}

func NewServer(imposterConfig *config.ImposterConfig, configDir string, configs []config.Config) *Server {
	return &Server{
		Addr:      ":" + imposterConfig.ServerPort,
		ConfigDir: configDir,
		Configs:   configs,
	}
}

func (s *Server) Start(imposterConfig *config.ImposterConfig) {
	fmt.Printf("Server is listening on %s...\n", s.Addr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(w, r, s.ConfigDir, s.Configs, imposterConfig)
	})

	if err := http.ListenAndServe(s.Addr, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
