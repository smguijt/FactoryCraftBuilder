package debug

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smguijt/factorycraftbuilder/internal/ctxkeys"
	"github.com/smguijt/factorycraftbuilder/internal/world"
	"github.com/smguijt/factorycraftbuilder/pkg/apierror"
)

// SnapshotLoader is satisfied by world.Service.
type SnapshotLoader interface {
	GetMapSnapshot(ctx context.Context, playerID, worldID string) (*world.MapSnapshot, error)
}

type Handler struct {
	svc         SnapshotLoader
	debugMapHTML []byte
}

func NewHandler(svc SnapshotLoader, debugMapHTML []byte) *Handler {
	return &Handler{svc: svc, debugMapHTML: debugMapHTML}
}

// GET /worlds/{worldID}/debug/map.svg
func (h *Handler) SVG(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	worldID := chi.URLParam(r, "worldID")

	snap, err := h.svc.GetMapSnapshot(r.Context(), playerID, worldID)
	if err != nil {
		if errors.Is(err, world.ErrNotFound) {
			apierror.Write(w, apierror.ErrNotFound)
			return
		}
		apierror.Write(w, apierror.ErrInternal)
		return
	}

	svg := GenerateSVG(snap)
	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(svg))
}

// GET /debug/map/{worldID}  — interactive HTML canvas viewer
// No auth on this endpoint; the page uses ?token= for the Firebase token.
func (h *Handler) HTMLViewer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.debugMapHTML)
}
