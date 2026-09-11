package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/karol/secretaria-escolar-backend/internal/application/health"
)

type Server struct {
	httpServer *http.Server
}

func New(addr string, healthService health.Service) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(healthService.Check())
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

func (server *Server) ListenAndServe() error {
	return server.httpServer.ListenAndServe()
}