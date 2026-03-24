package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/smguijt/factorycraftbuilder/internal/ctxkeys"
	"github.com/smguijt/factorycraftbuilder/pkg/apierror"
)

// Middleware verifies the Firebase ID token in the Authorization header.
// On success it injects the player UID and email into the request context.
func Middleware(client *auth.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				slog.Error("missing or invalid authorization header")
				apierror.Write(w, apierror.ErrUnauthorized)
				return
			}
			idToken := strings.TrimPrefix(header, "Bearer ")
			token, err := verifyToken(r.Context(), client, idToken)
			if err != nil {
				slog.Error("token verification failed", "error", err)
				apierror.Write(w, apierror.ErrUnauthorized)
				return
			}
			slog.Info("token verified", "playerID", token.UID, "email", token.Claims["email"])
			ctx := ctxkeys.WithPlayerID(r.Context(), token.UID)
			email, _ := token.Claims["email"].(string)
			ctx = ctxkeys.WithPlayerEmail(ctx, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// verifyToken checks if we're using the Firebase emulator, and if so, verifies
// the token against the emulator. Otherwise, uses the normal Firebase Admin SDK verification.
func verifyToken(ctx context.Context, client *auth.Client, idToken string) (*auth.Token, error) {
	emulatorHost := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if emulatorHost != "" {
		slog.Info("using Firebase Auth emulator", "host", emulatorHost)
		return verifyTokenWithEmulator(idToken, emulatorHost)
	}
	slog.Info("using Firebase Admin SDK for token verification")
	return client.VerifyIDToken(ctx, idToken)
}

// verifyTokenWithEmulator verifies an ID token using the Firebase Auth emulator.
func verifyTokenWithEmulator(idToken, emulatorHost string) (*auth.Token, error) {
	// Call the emulator's secureToken endpoint to decode the token
	url := fmt.Sprintf("http://%s/identitytoolkit.googleapis.com/v1/accounts:lookup?key=fake-api-key", emulatorHost)

	reqBody := map[string]interface{}{
		"idToken": idToken,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to call emulator: %w", err)
	}
	defer resp.Body.Close()

	// Read the entire response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("emulator returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Users []struct {
			LocalID string                 `json:"localId"`
			Email   string                 `json:"email"`
			Claims  map[string]interface{} `json:"customAttributes,omitempty"`
		} `json:"users"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode emulator response: %w", err)
	}

	if len(result.Users) == 0 {
		return nil, fmt.Errorf("token invalid or expired")
	}

	user := result.Users[0]
	claims := user.Claims
	if claims == nil {
		claims = make(map[string]interface{})
	}
	claims["email"] = user.Email

	return &auth.Token{
		UID:    user.LocalID,
		Claims: claims,
	}, nil
}
