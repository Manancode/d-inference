package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dginf/dginf/services/providerd/internal/runtime"
)

type Server struct {
	mux *http.ServeMux
}

func New(service *runtime.Service) *Server {
	mux := http.NewServeMux()
	server := &Server{mux: mux}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/v1/runtime/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, service.Health())
	})

	mux.HandleFunc("/v1/runtime/load-model", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		if err := service.LoadModel(request.Model); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, runtime.ErrModelRequired) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}

		writeJSON(w, http.StatusAccepted, service.Health())
	})

	return server
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
