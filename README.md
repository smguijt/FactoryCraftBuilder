# FactoryCraftBuilder

A factory/production simulation game engine built with Go and Google Cloud. Players create worlds, place buildings, build automated production chains, and advance through a research tree.

---

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Domain Concepts](#domain-concepts)
- [API Reference](#api-reference)
- [Authentication](#authentication)
- [Configuration](#configuration)
- [Simulation Engine](#simulation-engine)
- [Research Tree](#research-tree)
- [Error Handling](#error-handling)
- [Running Locally](#running-locally)

---

## Overview

FactoryCraftBuilder is a REST API that powers a factory-builder game. Players:

1. **Create Worlds** — Seeded 100×100 grids with ~300 resource nodes (iron, copper, stone, coal, gold, uranium, wolframite, oil, silicon)
2. **Place Buildings** — Extractors, conveyors, splitters, mergers, smelters, assemblers, chemical plants, research labs
3. **Connect Buildings** — Form production chains; extractors feed belts, belts feed factories
4. **Simulate Production** — Tick-based engine processes extraction, crafting, belt transport, and research
5. **Research Technologies** — Unlock new buildings and upgrades via a 14-node research tree
6. **Manage Inventory** — Track items, coins (economy), and research points

---

## Tech Stack

| Component       | Technology                         |
|-----------------|------------------------------------|
| Language        | Go 1.25                            |
| HTTP Router     | [Chi v5](https://github.com/go-chi/chi) |
| Database        | Cloud Firestore                    |
| Authentication  | Firebase Auth (JWT)                |
| Deployment      | Google Cloud Run                   |
| Local Dev       | Docker + Firebase Emulator Suite   |

---

## Project Structure

```
FactoryCraftBuilder/
├── cmd/server/main.go              # Entry point; wires router, middleware, handlers
├── internal/
│   ├── auth/                       # Firebase JWT middleware + login handler
│   ├── config/                     # Env var loading (godotenv)
│   ├── ctxkeys/                    # Shared context key constants
│   ├── debug/                      # SVG map visualisation (debug only)
│   ├── player/                     # Player model, service, repository, handler
│   ├── recipe/                     # Static game data types and registry
│   ├── research/                   # Research tree: types, service, repository, handler
│   ├── simulation/                 # Pure tick engine: extractors, belts, factories
│   ├── tick/                       # Tick orchestrator (load → simulate → persist)
│   └── world/                      # Worlds, buildings, nodes, inventory
├── pkg/
│   ├── apierror/                   # Typed JSON error responses
│   ├── firestore/                  # Firestore client initialisation
│   └── middleware/                 # Request logging + per-player rate limiting
├── static/
│   ├── recipes.json                # Items, recipes, extraction rates, building costs
│   └── research.json               # 14-node research tree definition
├── Docs/                           # Development phase notes + operational docs
├── Dockerfile                      # Multi-stage Alpine → distroless build
├── docker-compose.yml              # Local dev: server + Firebase emulators
├── firebase.json                   # Firebase CLI config (emulator ports)
└── .env.example                    # Configuration template
```

**Architecture pattern:** Handler → Service → Repository (Firestore), with a pure simulation engine called by the tick orchestrator.

---

## Domain Concepts

### Player
Authenticated user identified by Firebase UID.

| Field       | Type     | Description            |
|-------------|----------|------------------------|
| ID          | string   | Firebase UID           |
| Email       | string   |                        |
| DisplayName | string   |                        |
| CreatedAt   | time     |                        |

### World
A named game instance owned by a player. Holds a seeded 100×100 grid.

| Field           | Type     | Description                              |
|-----------------|----------|------------------------------------------|
| ID              | string   | UUID                                     |
| Name            | string   |                                          |
| Seed            | int64    | RNG seed for deterministic generation    |
| LastSimulatedAt | time     | Tracks offline play window               |

### ResourceNode
Fixed resource deposits on the world grid. An extractor must be placed on one to harvest it.

**Types:** `iron_mine`, `copper_mine`, `stone_mine`, `gold_mine`, `coal_mine`, `uranium`, `wolframite`, `oil_well`, `silicon_mine`, `forest`

### Building

| Field         | Type           | Description                                      |
|---------------|----------------|--------------------------------------------------|
| Type          | BuildingType   | See building types below                         |
| X, Y          | int            | Grid position                                    |
| Rotation      | int            | 0 / 90 / 180 / 270                               |
| RecipeID      | string         | Active recipe (factories only)                   |
| LinkedNodeID  | string         | Resource node (extractors only)                  |
| ExtractorTier | int            | 1–3; multiplies output (×1, ×2, ×3)             |
| InputSlots    | map[item]int64 | Input item buffers                               |
| OutputSlots   | map[item]int64 | Output item buffers (capped at 50 per item)      |
| Connections   | []string       | Ordered list of downstream building IDs          |
| IsActive      | bool           | Pauses production when false                     |

**Building types:**

| Type               | Purpose                                              |
|--------------------|------------------------------------------------------|
| `extractor`        | Harvests resources from a node                       |
| `conveyor`         | Transports items between buildings                   |
| `splitter`         | Distributes items round-robin (max 3 outputs)        |
| `merger`           | Aggregates items from multiple inputs (max 3 inputs) |
| `smelter`          | Basic crafting (iron bars, copper bars, etc.)        |
| `assembler`        | Mid-tier crafting (gears, circuits, etc.)            |
| `advanced_assembler` | High-tier crafting (requires research)             |
| `chemical_plant`   | Chemical recipes (requires research)                 |
| `research_lab`     | Converts items into coins and research points        |

### Inventory

| Field          | Type            | Description                              |
|----------------|-----------------|------------------------------------------|
| Items          | map[item]int64  | Current item counts                      |
| Coins          | int64           | Economy currency (placement + research)  |
| ResearchPoints | int64           | Unlock currency for the research tree    |
| TotalDelivered | map[item]int64  | Cumulative deliveries (research gates)   |

---

## API Reference

All endpoints under `/api/v1/` require `Authorization: Bearer <firebase-id-token>` except `/health`.

### Health
```
GET /health
→ { "status": "ok" }
```

### Static Game Data
```
GET /api/v1/recipes         → Recipe[]     (cached 1 hour)
GET /api/v1/items           → Item[]       (cached 1 hour)
GET /api/v1/research        → Node[]       (cached 1 hour)
GET /api/v1/buildings/{buildingType}/recipes → Recipe[]
```

### Authentication
```
POST /api/v1/auth/login
Header: Authorization: Bearer <token>
Body:   { "displayName": "string" }
→ Player{}    (creates player on first call, updates on subsequent)
```

### Players
```
GET   /api/v1/players/me        → Player{}
PATCH /api/v1/players/me        → Player{}
      Body: { "displayName": "string" }
```

### Worlds
```
GET    /api/v1/worlds                  → World[]
POST   /api/v1/worlds                  → World{}  (201)
       Body: { "name": "string" }
GET    /api/v1/worlds/{worldID}        → World{}
DELETE /api/v1/worlds/{worldID}        → 204
```

### Map & Simulation
```
GET  /api/v1/worlds/{worldID}/map      → MapSnapshot{}
     (auto-runs tick if world has been offline)
POST /api/v1/worlds/{worldID}/tick     → MapSnapshot{}
     (explicitly advance simulation to now)
```

`MapSnapshot` contains `world`, `resourceNodes`, `buildings`, and `inventory`.

### Resource Nodes
```
GET /api/v1/worlds/{worldID}/nodes              → ResourceNode[]
GET /api/v1/worlds/{worldID}/nodes/{nodeID}     → ResourceNode{}
```

### Buildings
```
GET    /api/v1/worlds/{worldID}/buildings                          → Building[]
POST   /api/v1/worlds/{worldID}/buildings                          → Building{}  (201)
       Body: { "type": "smelter", "x": 10, "y": 5, "rotation": 0 }
GET    /api/v1/worlds/{worldID}/buildings/{buildingID}             → Building{}
PATCH  /api/v1/worlds/{worldID}/buildings/{buildingID}             → Building{}
       Body: { "recipeID"?: "...", "isActive"?: true, "extractorTier"?: 2 }
DELETE /api/v1/worlds/{worldID}/buildings/{buildingID}             → 204
POST   /api/v1/worlds/{worldID}/buildings/{buildingID}/connect     → 204
       Body: { "targetID": "string" }
DELETE /api/v1/worlds/{worldID}/buildings/{buildingID}/connect/{targetID} → 204
```

### Inventory
```
GET /api/v1/worlds/{worldID}/inventory  → Inventory{}
```

### Research
```
GET  /api/v1/worlds/{worldID}/research                          → NodeProgress[]
POST /api/v1/worlds/{worldID}/research/{nodeID}/unlock          → WorldResearch{}
     (atomic: validates prerequisites, deducts coins/RP, records unlock)
```

### Debug (requires `DEBUG_ROUTES=true`)
```
GET /api/v1/worlds/{worldID}/debug/map.svg   → SVG (auth header)
GET /debug/map/{worldID}?token=<jwt>         → HTML viewer (token in query param)
```

---

## Authentication

1. The client authenticates with Firebase (outside this service) and receives a short-lived ID token.
2. Every request includes `Authorization: Bearer <idToken>`.
3. The auth middleware verifies the JWT using the Firebase Admin SDK (signature, expiry, audience).
4. On success the player's UID and email are injected into the request context.
5. The per-player rate limiter then allows up to **10 req/s** with a burst of **30**.

For local development with emulators, set `FIREBASE_AUTH_EMULATOR_HOST=localhost:9099`. The SDK skips real JWT validation against that emulator.

---

## Configuration

Copy `.env.example` to `.env` and fill in your values:

```env
# Server
PORT=8080

# Google Cloud
GCP_PROJECT_ID=your-gcp-project-id

# Firebase credentials — leave empty to use Application Default Credentials
FIREBASE_CREDS_PATH=./serviceAccountKey.json

# Game balance
STARTING_COINS=500
MAX_OFFLINE_SECONDS=28800   # 8 hours maximum offline simulation

# Enable debug endpoints — never true in production
DEBUG_ROUTES=true
```

For Docker / emulator usage, `GCP_PROJECT_ID` must be `demo-local` (the `demo-` prefix tells the Firebase SDK to use the emulator).

---

## Simulation Engine

The engine is a **pure function**: `Tick(state, now, deltaSeconds) → SimulationResult`. No side effects; all Firestore writes happen in the orchestrator after the tick.

**Tick steps:**
1. **Extractors** produce items proportional to elapsed time, node type, and tier. Output capped at 50 items per slot.
2. **Topological sort** builds the conveyor DAG using Kahn's algorithm (cycles are rejected at connection time).
3. **Belt transport** moves items along chains; splitters distribute round-robin; mergers aggregate.
4. **Factories** consume recipe inputs and produce outputs; multiple cycles are processed if buffers allow.
5. **Research lab** sells items for coins and research points at per-item rates.

Belt throughput scales with `BeltTier` (unlocked via research): Tier 1 = base, Tier 2 = 2×, Tier 3 = 3×.

---

## Research Tree

14 nodes in a branching tree. Each node has:

- **Prerequisites** — other nodes that must be unlocked first
- **Delivery requirements** — cumulative item counts that must have been processed through a research lab
- **Coin cost** and **Research Point cost** for the atomic unlock transaction

Unlocking a node grants one or more of: a new building type, a new recipe set, or a belt/extractor tier upgrade.

---

## Error Handling

All errors return JSON:
```json
{
  "code": "not_found",
  "message": "world not found"
}
```

| HTTP | Code                  | Meaning                                              |
|------|-----------------------|------------------------------------------------------|
| 400  | `bad_request`         | Occupied tile, invalid rotation, duplicate connection |
| 401  | `unauthorized`        | Missing or invalid Firebase token                    |
| 402  | `insufficient_funds`  | Not enough coins or research points                  |
| 403  | `forbidden`           | Building / recipe not yet researched                 |
| 404  | `not_found`           | World, building, or node does not exist              |
| 409  | `conflict`            | Research node already unlocked                       |
| 429  | `rate_limited`        | Per-player rate limit exceeded                       |
| 500  | `internal_error`      | Unexpected server or database error                  |

---

## Running Locally

See [Docs/DockerDesktop.md](Docs/DockerDesktop.md) for full Docker Desktop instructions and [Docs/TestingGuide.md](Docs/TestingGuide.md) for step-by-step testing.

### Quick start (without Docker)
```bash
cp .env.example .env
# Edit .env — set GCP_PROJECT_ID and FIREBASE_CREDS_PATH

go run ./cmd/server
# Server starts on :8080
```

### Run unit tests
```bash
go test ./internal/simulation/... -v
```
