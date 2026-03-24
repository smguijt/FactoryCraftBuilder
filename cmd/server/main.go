package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	firebase "firebase.google.com/go/v4"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"google.golang.org/api/option"

	"github.com/smguijt/factorycraftbuilder/internal/auth"
	"github.com/smguijt/factorycraftbuilder/internal/config"
	"github.com/smguijt/factorycraftbuilder/internal/player"
	fsClient "github.com/smguijt/factorycraftbuilder/pkg/firestore"
	appMiddleware "github.com/smguijt/factorycraftbuilder/pkg/middleware"
)

func main() {
	// Structured JSON logging — Cloud Logging parses this automatically
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	ctx := context.Background()

	// Firestore
	fs, err := fsClient.New(ctx, cfg.ProjectID, cfg.FirebaseCredsPath)
	if err != nil {
		slog.Error("failed to create firestore client", "error", err)
		os.Exit(1)
	}
	defer fs.Close()

	// Firebase Auth
	var fbOpts []option.ClientOption
	if cfg.FirebaseCredsPath != "" {
		fbOpts = append(fbOpts, option.WithCredentialsFile(cfg.FirebaseCredsPath))
	}
	fbApp, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, fbOpts...)
	if err != nil {
		slog.Error("failed to init firebase app", "error", err)
		os.Exit(1)
	}
	authClient, err := fbApp.Auth(ctx)
	if err != nil {
		slog.Error("failed to create firebase auth client", "error", err)
		os.Exit(1)
	}

	// Layers
	playerRepo := player.NewRepository(fs)
	playerSvc := player.NewService(playerRepo)
	playerHandler := player.NewHandler(playerSvc)
	authHandler := auth.NewHandler(playerSvc)

	// Router
	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(appMiddleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Auth (requires Firebase token in header)
		r.With(auth.Middleware(authClient)).Post("/auth/login", authHandler.Login)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(authClient))
			r.Get("/players/me", playerHandler.GetMe)
			r.Patch("/players/me", playerHandler.PatchMe)
		})
	})

	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
