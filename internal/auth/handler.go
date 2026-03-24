package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/smguijt/factorycraftbuilder/internal/ctxkeys"
	"github.com/smguijt/factorycraftbuilder/internal/player"
)

type Handler struct {
	players *player.Service
}

func NewHandler(players *player.Service) *Handler {
	return &Handler{players: players}
}

// POST /auth/login
// Requires Authorization: Bearer <firebase-id-token>.
// Upserts the player document and returns the player profile.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	playerID := ctxkeys.PlayerID(r.Context())
	email := ctxkeys.PlayerEmail(r.Context())

	slog.Info("login attempt", "playerID", playerID, "email", email)

	var body struct {
		DisplayName string `json:"displayName"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	p, err := h.players.Upsert(r.Context(), playerID, email, body.DisplayName)
	if err != nil {
		slog.Error("failed to upsert player", "playerID", playerID, "email", email, "error", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "internal_error",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}
