package api

import (
	"encoding/json"
	"net/http"

	"github.com/eigeninference/coordinator/internal/store"
)

// handleAdminUpsertModel handles POST /v1/admin/models.
// Adds a new model or updates an existing one in the catalog.
func (s *Server) handleAdminUpsertModel(w http.ResponseWriter, r *http.Request) {
	var model store.SupportedModel
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid_request_error", "invalid JSON: "+err.Error()))
		return
	}

	if model.ID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid_request_error", "id is required"))
		return
	}
	if model.ModelType == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid_request_error", "model_type is required"))
		return
	}

	if err := s.store.SetSupportedModel(&model); err != nil {
		s.logger.Error("failed to upsert model", "id", model.ID, "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse("internal_error", "failed to save model"))
		return
	}

	s.SyncModelCatalog()

	s.logger.Info("model catalog updated", "id", model.ID, "action", "upsert")
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"model": model,
	})
}

// handleAdminListCatalog handles GET /v1/admin/models.
// Returns the full model catalog including inactive models.
func (s *Server) handleAdminListCatalog(w http.ResponseWriter, r *http.Request) {
	models := s.store.ListSupportedModels()
	writeJSON(w, http.StatusOK, map[string]any{
		"models": models,
		"count":  len(models),
	})
}

// handleAdminRemoveModel handles DELETE /v1/admin/models?id=...
// Removes a model from the catalog.
func (s *Server) handleAdminRemoveModel(w http.ResponseWriter, r *http.Request) {
	modelID := r.URL.Query().Get("id")
	if modelID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse("invalid_request_error", "id query parameter is required"))
		return
	}

	if err := s.store.DeleteSupportedModel(modelID); err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse("not_found", "model not found: "+err.Error()))
		return
	}

	s.SyncModelCatalog()

	s.logger.Info("model catalog updated", "id", modelID, "action", "delete")
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"deleted": modelID,
	})
}
