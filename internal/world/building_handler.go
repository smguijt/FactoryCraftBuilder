package world

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smguijt/factorycraftbuilder/internal/ctxkeys"
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	"github.com/smguijt/factorycraftbuilder/pkg/apierror"
)

// POST /worlds/{worldID}/buildings
func (h *Handler) PlaceBuilding(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	var req PlaceBuildingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Write(w, apierror.ErrBadRequest("invalid JSON body"))
		return
	}

	b, err := h.svc.PlaceBuilding(r.Context(), playerID, worldID, req)
	if err != nil {
		writeJSON(w, buildingErrStatus(err), buildingErrBody(err))
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// GET /worlds/{worldID}/buildings
func (h *Handler) ListBuildings(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	buildings, err := h.svc.ListBuildings(r.Context(), playerID, worldID)
	if err != nil {
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, buildings)
}

// GET /worlds/{worldID}/buildings/{buildingID}
func (h *Handler) GetBuilding(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")
	buildingID := chi.URLParam(r, "buildingID")

	b, err := h.svc.GetBuilding(r.Context(), playerID, worldID, buildingID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// PATCH /worlds/{worldID}/buildings/{buildingID}
func (h *Handler) UpdateBuilding(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")
	buildingID := chi.URLParam(r, "buildingID")

	var req UpdateBuildingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.Write(w, apierror.ErrBadRequest("invalid JSON body"))
		return
	}

	b, err := h.svc.UpdateBuilding(r.Context(), playerID, worldID, buildingID, req)
	if err != nil {
		writeJSON(w, buildingErrStatus(err), buildingErrBody(err))
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// DELETE /worlds/{worldID}/buildings/{buildingID}
func (h *Handler) DeleteBuilding(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")
	buildingID := chi.URLParam(r, "buildingID")

	if err := h.svc.DeleteBuilding(r.Context(), playerID, worldID, buildingID); err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /worlds/{worldID}/buildings/{buildingID}/connect
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")
	buildingID := chi.URLParam(r, "buildingID")

	var body struct {
		TargetID string `json:"targetID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TargetID == "" {
		apierror.Write(w, apierror.ErrBadRequest("targetID is required"))
		return
	}

	if err := h.svc.Connect(r.Context(), playerID, worldID, buildingID, body.TargetID); err != nil {
		writeJSON(w, buildingErrStatus(err), buildingErrBody(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /worlds/{worldID}/buildings/{buildingID}/connect/{targetID}
func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")
	buildingID := chi.URLParam(r, "buildingID")
	targetID := chi.URLParam(r, "targetID")

	if err := h.svc.Disconnect(r.Context(), playerID, worldID, buildingID, targetID); err != nil {
		writeJSON(w, buildingErrStatus(err), buildingErrBody(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// buildingErrStatus maps domain errors to HTTP status codes.
func buildingErrStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrResearchLocked),
		errors.Is(err, ErrTierNotUnlocked):
		return http.StatusForbidden
	case errors.Is(err, ErrOccupied),
		errors.Is(err, ErrNoNodeHere),
		errors.Is(err, ErrNodeTaken),
		errors.Is(err, ErrOutOfBounds),
		errors.Is(err, ErrInvalidRotation),
		errors.Is(err, ErrUnknownBuilding),
		errors.Is(err, ErrUnknownRecipe),
		errors.Is(err, ErrWrongFactory),
		errors.Is(err, ErrCycleDetected),
		errors.Is(err, ErrTooManyOutputs),
		errors.Is(err, ErrTooManyInputs),
		errors.Is(err, ErrAlreadyConnected),
		errors.Is(err, ErrSelfConnect),
		errors.Is(err, ErrConnNotFound),
		errors.Is(err, ErrNotAnExtractor),
		errors.Is(err, ErrInvalidExtractorTier):
		return http.StatusBadRequest
	case errors.Is(err, ErrInsufficientFunds):
		return http.StatusPaymentRequired
	default:
		return http.StatusInternalServerError
	}
}

func buildingErrBody(err error) *apierror.APIError {
	switch {
	case errors.Is(err, ErrNotFound):
		return apierror.ErrNotFound
	case errors.Is(err, ErrInsufficientFunds):
		return apierror.ErrInsufficientFunds(err.Error())
	case errors.Is(err, ErrResearchLocked):
		return apierror.New(http.StatusForbidden, "research_locked", err.Error())
	case errors.Is(err, ErrTierNotUnlocked):
		return apierror.New(http.StatusForbidden, "tier_not_unlocked", err.Error())
	case errors.Is(err, ErrOccupied),
		errors.Is(err, ErrNoNodeHere),
		errors.Is(err, ErrNodeTaken),
		errors.Is(err, ErrOutOfBounds),
		errors.Is(err, ErrInvalidRotation),
		errors.Is(err, ErrUnknownBuilding),
		errors.Is(err, ErrUnknownRecipe),
		errors.Is(err, ErrWrongFactory),
		errors.Is(err, ErrCycleDetected),
		errors.Is(err, ErrTooManyOutputs),
		errors.Is(err, ErrTooManyInputs),
		errors.Is(err, ErrAlreadyConnected),
		errors.Is(err, ErrSelfConnect),
		errors.Is(err, ErrConnNotFound),
		errors.Is(err, ErrNotAnExtractor),
		errors.Is(err, ErrInvalidExtractorTier):
		return apierror.ErrBadRequest(err.Error())
	default:
		return apierror.ErrInternal
	}
}

// RecipesForBuilding returns the recipes valid for a given building type.
// GET /worlds/{worldID}/buildings/{buildingID}/recipes  (convenience endpoint)
func (h *Handler) RecipesForBuilding(w http.ResponseWriter, r *http.Request) {
	buildingType := recipe.BuildingType(chi.URLParam(r, "buildingType"))
	var matching []string
	for _, rec := range h.svc.Registry().Recipes {
		if rec.FactoryType == buildingType {
			matching = append(matching, rec.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"buildingType": buildingType,
		"recipeIDs":    matching,
	})
}
