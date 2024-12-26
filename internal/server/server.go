package server

import (
	"fmt"
	"github.com/gatehill/imposter-go/internal/handler"
	"net/http"
)

type Server struct {
	Addr      string
	ConfigDir string
	Resources []handler.Resource
}

func NewServer(configDir string, resources []handler.Resource) *Server {
	return &Server{
		Addr:      ":8080",
		ConfigDir: configDir,
		Resources: resources,
	}
}

func (s *Server) Start() {
	fmt.Printf("Server is listening on %s...\n", s.Addr)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(w, r, s.ConfigDir, s.Resources)
	})

	if err := http.ListenAndServe(s.Addr, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
