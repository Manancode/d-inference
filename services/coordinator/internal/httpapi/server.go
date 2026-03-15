package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dginf/dginf/services/coordinator/internal/auth"
	"github.com/dginf/dginf/services/coordinator/internal/catalog"
)

type Server struct {
	mux *http.ServeMux
}

func New(challenges *auth.ChallengeService, entries []catalog.Entry) *Server {
	mux := http.NewServeMux()
	server := &Server{mux: mux}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"models": entries})
	})

	mux.HandleFunc("/v1/auth/challenge", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Wallet string `json:"wallet"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		challenge, err := challenges.New(request.Wallet)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, auth.ErrWalletRequired) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}

		writeJSON(w, http.StatusOK, challenge)
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
