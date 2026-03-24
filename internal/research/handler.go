package research

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smguijt/factorycraftbuilder/internal/ctxkeys"
	"github.com/smguijt/factorycraftbuilder/pkg/apierror"
)

// InventoryLoader is a function that returns totalDelivered for a world.
// Injected at wire-up time to avoid importing the world package.
type InventoryLoader func(ctx context.Context, playerID, worldID string) (map[string]int64, error)

type Handler struct {
	svc         *Service
	loadInvFn   InventoryLoader
}

func NewHandler(svc *Service, loadInvFn InventoryLoader) *Handler {
	return &Handler{svc: svc, loadInvFn: loadInvFn}
}

// GET /research — full static tree (cacheable, same for all players)
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_ = json.NewEncoder(w).Encode(h.svc.GetTree())
}

// GET /worlds/{worldID}/research
func (h *Handler) GetWorldResearch(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	totalDelivered, err := h.loadInvFn(r.Context(), playerID, worldID)
	if err != nil {
		apierror.Write(w, apierror.ErrInternal)
		return
	}

	progress, err := h.svc.GetWorldProgress(r.Context(), playerID, worldID, totalDelivered)
	if err != nil {
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

// POST /worlds/{worldID}/research/{nodeID}/unlock
func (h *Handler) UnlockNode(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")
	nodeID := chi.URLParam(r, "nodeID")

	wr, err := h.svc.UnlockNode(r.Context(), playerID, worldID, nodeID)
	if err != nil {
		writeJSON(w, researchErrStatus(err), researchErrBody(err))
		return
	}
	writeJSON(w, http.StatusOK, wr)
}

func researchErrStatus(err error) int {
	switch {
	case errors.Is(err, ErrUnknownNode):
		return http.StatusNotFound
	case errors.Is(err, ErrAlreadyUnlocked),
		errors.Is(err, ErrPrereqNotMet),
		errors.Is(err, ErrDeliveryNotMet):
		return http.StatusBadRequest
	case errors.Is(err, ErrInsufficientRP),
		errors.Is(err, ErrInsufficientCoins):
		return http.StatusPaymentRequired
	default:
		return http.StatusInternalServerError
	}
}

func researchErrBody(err error) *apierror.APIError {
	switch {
	case errors.Is(err, ErrUnknownNode):
		return apierror.ErrNotFound
	case errors.Is(err, ErrInsufficientRP), errors.Is(err, ErrInsufficientCoins):
		return apierror.ErrInsufficientFunds(err.Error())
	case errors.Is(err, ErrAlreadyUnlocked),
		errors.Is(err, ErrPrereqNotMet),
		errors.Is(err, ErrDeliveryNotMet):
		return apierror.ErrBadRequest(err.Error())
	default:
		return apierror.ErrInternal
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
