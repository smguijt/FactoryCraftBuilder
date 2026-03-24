go.mod + go.sum — module github.com/smguijt/factorycraftbuilder
cmd/server/main.go — Chi router, health endpoint, wires all layers
internal/config/config.go — env-based config with defaults
internal/ctxkeys/ctxkeys.go — shared context key helpers
internal/auth/middleware.go — Firebase JWT verification
internal/auth/handler.go — POST /auth/login
internal/player/ — player model, repository (Firestore), service, handler (GET/PATCH /players/me)
pkg/firestore/client.go — ADC-based client
pkg/middleware/logging.go — structured JSON request logging
pkg/apierror/apierror.go — typed API errors
Dockerfile — multi-stage distroless build