<li>go.mod + go.sum — module github.com/smguijt/factorycraftbuilder</li>
<li>cmd/server/main.go — Chi router, health endpoint, wires all layers</li>
<li>internal/config/config.go — env-based config with defaults</li>
<li>internal/ctxkeys/ctxkeys.go — shared context key helpers</li>
<li>internal/auth/middleware.go — Firebase JWT verification</li>
<li>internal/auth/handler.go — POST /auth/login</li>
<li>internal/player/ — player model, repository (Firestore), service, handler (GET/PATCH /players/me)</li>
<li>pkg/firestore/client.go — ADC-based client</li>
<li>pkg/middleware/logging.go — structured JSON request logging</li>
<li>pkg/apierror/apierror.go — typed API errors</li>
<li>Dockerfile — multi-stage distroless build</li>