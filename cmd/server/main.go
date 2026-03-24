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
	"github.com/smguijt/factorycraftbuilder/internal/recipe"
	researchPkg "github.com/smguijt/factorycraftbuilder/internal/research"
	"github.com/smguijt/factorycraftbuilder/internal/tick"
	"github.com/smguijt/factorycraftbuilder/internal/world"
	fsClient "github.com/smguijt/factorycraftbuilder/pkg/firestore"
	appMiddleware "github.com/smguijt/factorycraftbuilder/pkg/middleware"
	"github.com/smguijt/factorycraftbuilder/static"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	// Recipe registry
	recipeReg, err := recipe.LoadRegistry(static.RecipesJSON)
	if err != nil {
		slog.Error("failed to load recipes.json", "error", err)
		os.Exit(1)
	}
	slog.Info("recipe registry loaded", "recipes", len(recipeReg.Recipes), "items", len(recipeReg.ItemByID))

	// Research registry
	researchReg, err := researchPkg.LoadRegistry(static.ResearchJSON)
	if err != nil {
		slog.Error("failed to load research.json", "error", err)
		os.Exit(1)
	}
	slog.Info("research registry loaded", "nodes", len(researchReg.Nodes))

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

	// Player layer
	playerRepo := player.NewRepository(fs)
	playerSvc := player.NewService(playerRepo)
	playerHandler := player.NewHandler(playerSvc)
	authHandler := auth.NewHandler(playerSvc)

	// World layer
	worldRepo := world.NewRepository(fs)
	worldSvc := world.NewService(worldRepo, recipeReg, cfg.StartingCoins)

	// Research layer — depends on worldRepo.InventoryRef for atomic unlock transactions
	researchRepo := researchPkg.NewRepository(fs)
	researchSvc := researchPkg.NewService(researchRepo, researchReg, worldRepo.InventoryRef)
	researchHandler := researchPkg.NewHandler(researchSvc, func(ctx context.Context, playerID, worldID string) (map[string]int64, error) {
		inv, err := worldSvc.GetInventory(ctx, playerID, worldID)
		if err != nil {
			return nil, err
		}
		return inv.TotalDelivered, nil
	})

	// Wire research checker into world service (breaks the init cycle)
	worldSvc.SetResearchChecker(researchSvc)

	// Tick orchestrator + belt-tier injection
	tickOrchestrator := tick.New(worldRepo, recipeReg, fs, cfg.MaxOfflineSeconds)
	tickOrchestrator.SetBeltTierFn(func(ctx context.Context, playerID, worldID string) (int, error) {
		wr, err := researchSvc.GetState(ctx, playerID, worldID)
		if err != nil {
			return 1, err
		}
		return wr.BeltTier, nil
	})

	worldHandler := world.NewHandler(worldSvc, tickOrchestrator)

	// Router
	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(appMiddleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.With(auth.Middleware(authClient)).Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(authClient))

			// Players
			r.Get("/players/me", playerHandler.GetMe)
			r.Patch("/players/me", playerHandler.PatchMe)

			// Worlds
			r.Get("/worlds", worldHandler.ListWorlds)
			r.Post("/worlds", worldHandler.CreateWorld)
			r.Get("/worlds/{worldID}", worldHandler.GetWorld)
			r.Delete("/worlds/{worldID}", worldHandler.DeleteWorld)
			r.Get("/worlds/{worldID}/map", worldHandler.GetMap)
			r.Post("/worlds/{worldID}/tick", worldHandler.Tick)

			// Nodes
			r.Get("/worlds/{worldID}/nodes", worldHandler.ListNodes)
			r.Get("/worlds/{worldID}/nodes/{nodeID}", worldHandler.GetNode)

			// Buildings
			r.Get("/worlds/{worldID}/buildings", worldHandler.ListBuildings)
			r.Post("/worlds/{worldID}/buildings", worldHandler.PlaceBuilding)
			r.Get("/worlds/{worldID}/buildings/{buildingID}", worldHandler.GetBuilding)
			r.Patch("/worlds/{worldID}/buildings/{buildingID}", worldHandler.UpdateBuilding)
			r.Delete("/worlds/{worldID}/buildings/{buildingID}", worldHandler.DeleteBuilding)
			r.Post("/worlds/{worldID}/buildings/{buildingID}/connect", worldHandler.Connect)
			r.Delete("/worlds/{worldID}/buildings/{buildingID}/connect/{targetID}", worldHandler.Disconnect)

			// Convenience: recipes filtered by building type
			r.Get("/buildings/{buildingType}/recipes", worldHandler.RecipesForBuilding)

			// Inventory
			r.Get("/worlds/{worldID}/inventory", worldHandler.GetInventory)

			// Research
			r.Get("/research", researchHandler.GetTree)
			r.Get("/worlds/{worldID}/research", researchHandler.GetWorldResearch)
			r.Post("/worlds/{worldID}/research/{nodeID}/unlock", researchHandler.UnlockNode)

			// Static game data (cacheable)
			r.Get("/recipes", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "public, max-age=3600")
				_, _ = w.Write(static.RecipesJSON)
			})
		})
	})

	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
