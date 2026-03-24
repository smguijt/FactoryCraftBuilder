package world

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smguijt/factorycraftbuilder/internal/ctxkeys"
	"github.com/smguijt/factorycraftbuilder/pkg/apierror"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /worlds
func (h *Handler) ListWorlds(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worlds, err := h.svc.ListWorlds(r.Context(), playerID)
	if err != nil {
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, worlds)
}

// POST /worlds
func (h *Handler) CreateWorld(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		apierror.Write(w, apierror.ErrBadRequest("name is required"))
		return
	}

	world, err := h.svc.CreateWorld(r.Context(), playerID, body.Name)
	if err != nil {
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusCreated, world)
}

// GET /worlds/{worldID}
func (h *Handler) GetWorld(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	world, err := h.svc.GetWorld(r.Context(), playerID, worldID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, world)
}

// DELETE /worlds/{worldID}
func (h *Handler) DeleteWorld(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	if err := h.svc.DeleteWorld(r.Context(), playerID, worldID); err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /worlds/{worldID}/map
func (h *Handler) GetMap(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	snap, err := h.svc.GetMapSnapshot(r.Context(), playerID, worldID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// GET /worlds/{worldID}/nodes
func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	nodes, err := h.svc.ListNodes(r.Context(), playerID, worldID)
	if err != nil {
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

// GET /worlds/{worldID}/nodes/{nodeID}
func (h *Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")
	nodeID := chi.URLParam(r, "nodeID")

	node, err := h.svc.GetNode(r.Context(), playerID, worldID, nodeID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

// GET /worlds/{worldID}/inventory
func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	inv, err := h.svc.GetInventory(r.Context(), playerID, worldID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// POST /worlds/{worldID}/tick — placeholder; implemented in Phase 4.
func (h *Handler) Tick(w http.ResponseWriter, r *http.Request) {
	apierror.Write(w, apierror.New(http.StatusNotImplemented, "not_implemented", "tick not yet implemented"))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
