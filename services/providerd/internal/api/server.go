package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dginf/dginf/services/providerd/internal/app"
	"github.com/dginf/dginf/services/providerd/internal/domain"
)

type Server struct {
	service *app.Service
	mux     *http.ServeMux
}

func NewServer(service *app.Service) *Server {
	server := &Server{
		service: service,
		mux:     http.NewServeMux(),
	}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /v1/status", s.handleStatus)
	s.mux.HandleFunc("POST /v1/pause", s.handlePause)
	s.mux.HandleFunc("POST /v1/resume", s.handleResume)
	s.mux.HandleFunc("POST /v1/runtime/load-selected-model", s.handleLoadSelectedModel)
	s.mux.HandleFunc("POST /v1/jobs/execute", s.handleExecuteJob)
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.service.Status())
}

func (s *Server) handlePause(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.service.Pause())
}

func (s *Server) handleResume(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.service.Resume())
}

func (s *Server) handleLoadSelectedModel(w http.ResponseWriter, r *http.Request) {
	if err := s.service.LoadSelectedModel(r.Context()); err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "loaded"})
}

func (s *Server) handleExecuteJob(w http.ResponseWriter, r *http.Request) {
	var req domain.ExecuteJobRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.service.ExecuteJob(r.Context(), req)
	if err != nil {
		mapError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrNodeBusy), errors.Is(err, app.ErrNodePaused):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, app.ErrNoRuntime):
		writeError(w, http.StatusFailedDependency, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}
