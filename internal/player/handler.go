package player

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/smguijt/factorycraftbuilder/internal/ctxkeys"
	"github.com/smguijt/factorycraftbuilder/pkg/apierror"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /players/me
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	p, err := h.svc.Get(r.Context(), playerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

// PATCH /players/me
func (h *Handler) PatchMe(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	email := ctxkeys.PlayerEmail(r.Context())

	var body struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apierror.Write(w, apierror.ErrBadRequest("invalid JSON body"))
		return
	}

	p, err := h.svc.Upsert(r.Context(), playerID, email, body.DisplayName)
	if err != nil {
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}
