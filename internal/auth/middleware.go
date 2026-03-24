package auth

import (
	"net/http"
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
				apierror.Write(w, apierror.ErrUnauthorized)
				return
			}
			idToken := strings.TrimPrefix(header, "Bearer ")
			token, err := client.VerifyIDToken(r.Context(), idToken)
			if err != nil {
				apierror.Write(w, apierror.ErrUnauthorized)
				return
			}
			ctx := ctxkeys.WithPlayerID(r.Context(), token.UID)
			email, _ := token.Claims["email"].(string)
			ctx = ctxkeys.WithPlayerEmail(ctx, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
