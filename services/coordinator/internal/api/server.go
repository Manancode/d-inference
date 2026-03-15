package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dginf/dginf/services/coordinator/internal/app"
	"github.com/dginf/dginf/services/coordinator/internal/domain"
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
	s.mux.HandleFunc("POST /v1/auth/challenge", s.handleChallenge)
	s.mux.HandleFunc("POST /v1/auth/verify", s.handleVerify)
	s.mux.HandleFunc("POST /v1/providers/register", s.handleRegisterProvider)
	s.mux.HandleFunc("POST /v1/providers/heartbeat", s.handleProviderHeartbeat)
	s.mux.HandleFunc("GET /v1/providers", s.handleProviders)
	s.mux.HandleFunc("GET /v1/models", s.handleModels)
	s.mux.HandleFunc("POST /v1/jobs/quote", s.handleQuote)
	s.mux.HandleFunc("POST /v1/jobs", s.handleCreateJob)
	s.mux.HandleFunc("GET /v1/jobs/{jobId}", s.handleJob)
	s.mux.HandleFunc("POST /v1/jobs/{jobId}/complete", s.handleCompleteJob)
	s.mux.HandleFunc("POST /v1/jobs/{jobId}/run", s.handleRunJob)
	s.mux.HandleFunc("GET /v1/jobs/{jobId}/settlement-voucher", s.handleSettlementVoucher)
	s.mux.HandleFunc("POST /v1/jobs/{jobId}/cancel", s.handleCancelJob)
	s.mux.HandleFunc("GET /v1/balances", s.handleBalances)
	s.mux.HandleFunc("POST /v1/dev/seed-balance", s.handleSeedBalance)
}

func (s *Server) handleChallenge(w http.ResponseWriter, r *http.Request) {
	var req domain.AuthChallengeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response, err := s.service.IssueChallenge(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req domain.AuthVerifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response, err := s.service.VerifyChallenge(req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleRegisterProvider(w http.ResponseWriter, r *http.Request) {
	var req domain.ProviderRegistration
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.service.RegisterProvider(req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleProviderHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req domain.ProviderHeartbeat
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.service.Heartbeat(req); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleModels(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"models": s.service.Models()})
}

func (s *Server) handleProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"providers": s.service.Providers()})
}

func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	var req domain.JobQuoteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response, err := s.service.QuoteJob(req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req domain.JobCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response, err := s.service.CreateJob(req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.service.Job(r.PathValue("jobId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleCompleteJob(w http.ResponseWriter, r *http.Request) {
	var req domain.JobCompletionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	job, err := s.service.CompleteJob(r.PathValue("jobId"), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleRunJob(w http.ResponseWriter, r *http.Request) {
	var req domain.JobRunRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.service.RunJob(r.PathValue("jobId"), req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSettlementVoucher(w http.ResponseWriter, r *http.Request) {
	response, err := s.service.SettlementVoucher(r.PathValue("jobId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if err := s.service.CancelJob(r.PathValue("jobId")); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleBalances(w http.ResponseWriter, r *http.Request) {
	wallet := r.URL.Query().Get("wallet")
	if wallet == "" {
		writeError(w, http.StatusBadRequest, errors.New("wallet query parameter is required"))
		return
	}
	writeJSON(w, http.StatusOK, s.service.Balance(wallet))
}

func (s *Server) handleSeedBalance(w http.ResponseWriter, r *http.Request) {
	var req domain.SeedBalanceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.service.SeedWalletBalance(req))
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrInsufficientFunds:
		writeError(w, http.StatusPaymentRequired, err)
	case domain.ErrModelUnavailable, domain.ErrNoCapacity:
		writeError(w, http.StatusConflict, err)
	case domain.ErrQuoteNotFound, domain.ErrJobNotFound, domain.ErrChallengeNotFound:
		writeError(w, http.StatusNotFound, err)
	case domain.ErrQuoteExpired, domain.ErrChallengeExpired:
		writeError(w, http.StatusGone, err)
	case domain.ErrQuoteConsumed, domain.ErrChallengeMismatch, domain.ErrInvalidSignature:
		writeError(w, http.StatusUnauthorized, err)
	case domain.ErrJobNotCompletable:
		writeError(w, http.StatusConflict, err)
	case domain.ErrProviderUnreachable:
		writeError(w, http.StatusBadGateway, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}
