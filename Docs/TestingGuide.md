# Testing Guide — FactoryCraftBuilder

This guide walks through testing the entire FactoryCraftBuilder API from scratch: from getting an auth token to running a full production simulation. Follow the steps in order.

---

## Prerequisites

- The Docker stack is running (`docker compose up --build`)
- `curl` and `jq` available in your terminal (`brew install jq` on macOS)
- API is responding: `curl http://localhost:8080/health` returns `{"status":"ok"}`

---

## Step 1 — Create a Test User

Open the Firebase Emulator UI at http://localhost:4000 and go to **Authentication**.

1. Click **Add user**
2. Set **Email:** `test@example.com`
3. Set **Password:** `Test1234!`
4. Click **Save**

---

## Step 2 — Obtain an ID Token

The Firebase Auth emulator exposes a REST sign-in endpoint. Run:

```bash
TOKEN=$(curl -s -X POST \
  "http://localhost:9099/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=fake-api-key" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test1234!","returnSecureToken":true}' \
  | jq -r '.idToken')

echo "Token: $TOKEN"
```

Save the token — it is used as `Bearer $TOKEN` in all subsequent requests.

---

## Step 3 — Login / Create Player Profile

The login endpoint upserts a player document for the authenticated user.

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"displayName":"Test Player"}' | jq
```

**Expected response:**
```json
{
  "id": "<firebase-uid>",
  "email": "test@example.com",
  "displayName": "Test Player",
  "createdAt": "..."
}
```

**What to verify:**
- HTTP 200
- `id` matches the Firebase UID shown in the Emulator UI under Authentication
- Player document is visible in Firestore UI at http://localhost:4000 under `players/{uid}`

---

## Step 4 — Fetch Static Game Data

These endpoints return the full recipe and research catalogues and do not modify state.

### Fetch all items
```bash
curl -s http://localhost:8080/api/v1/items \
  -H "Authorization: Bearer $TOKEN" | jq 'length'
```
Expected: a number greater than 0 (typically 20+).

### Fetch all recipes
```bash
curl -s http://localhost:8080/api/v1/recipes \
  -H "Authorization: Bearer $TOKEN" | jq '[.[].id]'
```
Expected: an array of recipe IDs such as `iron_bar`, `copper_bar`, `gear`, etc.

### Fetch the research tree
```bash
curl -s http://localhost:8080/api/v1/research \
  -H "Authorization: Bearer $TOKEN" | jq '[.[].name]'
```
Expected: 14 research node names.

---

## Step 5 — Create a World

```bash
WORLD=$(curl -s -X POST http://localhost:8080/api/v1/worlds \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test World"}' | jq '.')

echo "$WORLD" | jq
WORLD_ID=$(echo "$WORLD" | jq -r '.id')
echo "World ID: $WORLD_ID"
```

**Expected response:**
```json
{
  "id": "<uuid>",
  "name": "Test World",
  "seed": <number>,
  "width": 100,
  "height": 100,
  "createdAt": "...",
  "lastSimulatedAt": "..."
}
```

**What to verify:**
- HTTP 201
- `width` and `height` are 100
- World document appears in Firestore UI under `players/{uid}/worlds/{worldID}`

---

## Step 6 — Explore the World Map

```bash
curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/map \
  -H "Authorization: Bearer $TOKEN" | jq '{
    nodeCount: (.resourceNodes | length),
    buildingCount: (.buildings | length),
    coins: .inventory.coins
  }'
```

**Expected:**
```json
{
  "nodeCount": <number around 300>,
  "buildingCount": 0,
  "coins": 500
}
```

**What to verify:**
- `nodeCount` is around 300 (seeded random placement)
- `coins` equals `STARTING_COINS` from your `.env` (default 500)
- `buildings` is empty — no buildings placed yet

### Find an iron mine node to use later

```bash
IRON_NODE=$(curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/nodes \
  -H "Authorization: Bearer $TOKEN" \
  | jq '[.[] | select(.type == "iron_mine")] | first')

echo "$IRON_NODE" | jq
NODE_ID=$(echo "$IRON_NODE" | jq -r '.id')
NODE_X=$(echo "$IRON_NODE" | jq -r '.x')
NODE_Y=$(echo "$IRON_NODE" | jq -r '.y')
echo "Iron node at ($NODE_X, $NODE_Y)"
```

---

## Step 7 — Place an Extractor

Place an extractor on the iron mine node found above.

```bash
EXTRACTOR=$(curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"type\":\"extractor\",\"x\":$NODE_X,\"y\":$NODE_Y,\"rotation\":0}" | jq '.')

echo "$EXTRACTOR" | jq
EXTRACTOR_ID=$(echo "$EXTRACTOR" | jq -r '.id')
echo "Extractor ID: $EXTRACTOR_ID"
```

**Expected:**
- HTTP 201
- `linkedNodeID` matches `$NODE_ID`
- `linkedNodeType` is `iron_mine`
- `isActive` is `true`

### Error cases to test

**Place on the same tile again (should fail):**
```bash
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"type\":\"extractor\",\"x\":$NODE_X,\"y\":$NODE_Y,\"rotation\":0}" | jq
```
Expected: HTTP 400, `"code": "bad_request"`, message about tile being occupied.

**Place extractor off a resource node (should fail):**
```bash
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"extractor","x":0,"y":0,"rotation":0}' | jq
```
Expected: HTTP 400, `"code": "bad_request"`, message about no node at that position.

---

## Step 8 — Place a Conveyor and a Smelter

Find an empty tile adjacent to the extractor, then place a smelter a few tiles away.

```bash
# Place conveyor one tile to the right of the extractor
CONVEYOR_X=$((NODE_X + 1))
CONVEYOR_Y=$NODE_Y

CONVEYOR=$(curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"type\":\"conveyor\",\"x\":$CONVEYOR_X,\"y\":$CONVEYOR_Y,\"rotation\":0}" | jq '.')

CONVEYOR_ID=$(echo "$CONVEYOR" | jq -r '.id')
echo "Conveyor ID: $CONVEYOR_ID"

# Place smelter two tiles to the right
SMELTER_X=$((NODE_X + 2))
SMELTER_Y=$NODE_Y

SMELTER=$(curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"type\":\"smelter\",\"x\":$SMELTER_X,\"y\":$SMELTER_Y,\"rotation\":0}" | jq '.')

SMELTER_ID=$(echo "$SMELTER" | jq -r '.id')
echo "Smelter ID: $SMELTER_ID"
```

**What to verify:**
- Both return HTTP 201
- Coin balance decreased (smelter has a placement cost)

Check updated coin balance:
```bash
curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/inventory \
  -H "Authorization: Bearer $TOKEN" | jq '.coins'
```

---

## Step 9 — Set a Recipe on the Smelter

```bash
# View available recipes for smelter
curl -s http://localhost:8080/api/v1/buildings/smelter/recipes \
  -H "Authorization: Bearer $TOKEN" | jq '[.[].id]'
```

Assign the `iron_bar` recipe:
```bash
curl -s -X PATCH \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$SMELTER_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"recipeID":"iron_bar"}' | jq '.recipeID'
```
Expected: `"iron_bar"`

---

## Step 10 — Connect the Buildings

Connect: **Extractor → Conveyor → Smelter**

```bash
# Connect extractor to conveyor
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$EXTRACTOR_ID/connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"targetID\":\"$CONVEYOR_ID\"}"

echo "Extractor → Conveyor: $?"

# Connect conveyor to smelter
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$CONVEYOR_ID/connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"targetID\":\"$SMELTER_ID\"}"

echo "Conveyor → Smelter: $?"
```

Both should return HTTP 204 (no body).

### Error case — cycle detection

Try connecting the smelter back to the extractor (should fail):
```bash
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$SMELTER_ID/connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"targetID\":\"$EXTRACTOR_ID\"}" | jq
```
Expected: HTTP 400, `"code": "bad_request"`, message about cycle detected.

---

## Step 11 — Run a Simulation Tick

Advance the simulation to the current time:

```bash
curl -s -X POST http://localhost:8080/api/v1/worlds/$WORLD_ID/tick \
  -H "Authorization: Bearer $TOKEN" | jq '{
    coins: .inventory.coins,
    researchPoints: .inventory.researchPoints,
    items: .inventory.items
  }'
```

**What to verify:**
- `inventory.items` now contains some `iron_ore` (raw) or `iron_bar` (if smelter had time to process)
- Building `outputSlots` are non-empty — check via:

```bash
curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$EXTRACTOR_ID \
  -H "Authorization: Bearer $TOKEN" | jq '.outputSlots'
```

---

## Step 12 — Place a Research Lab and Generate Research Points

```bash
# Find an empty tile
LAB_X=$((NODE_X + 4))
LAB_Y=$NODE_Y

LAB=$(curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"type\":\"research_lab\",\"x\":$LAB_X,\"y\":$LAB_Y,\"rotation\":0}" | jq '.')

LAB_ID=$(echo "$LAB" | jq -r '.id')
echo "Research Lab ID: $LAB_ID"
```

Connect the smelter output to the research lab:
```bash
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$SMELTER_ID/connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"targetID\":\"$LAB_ID\"}"
```

Run several ticks and watch research points accumulate:
```bash
for i in 1 2 3; do
  echo "--- Tick $i ---"
  curl -s -X POST http://localhost:8080/api/v1/worlds/$WORLD_ID/tick \
    -H "Authorization: Bearer $TOKEN" \
    | jq '{coins: .inventory.coins, rp: .inventory.researchPoints}'
  sleep 1
done
```

---

## Step 13 — Check Research Progress

```bash
curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/research \
  -H "Authorization: Bearer $TOKEN" | jq '[.[] | {name: .name, canUnlock: .canUnlock, isUnlocked: .isUnlocked}]'
```

**What to verify:**
- Nodes show `"isUnlocked": false` initially
- `deliveryProgress` fields populate as items are processed through the research lab
- `canUnlock` becomes `true` once all delivery requirements and costs are met

---

## Step 14 — Unlock a Research Node

Find a node that `canUnlock` is `true` (after accumulating enough RP and deliveries):

```bash
UNLOCKABLE_NODE=$(curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/research \
  -H "Authorization: Bearer $TOKEN" \
  | jq '[.[] | select(.canUnlock == true)] | first')

NODE_NAME=$(echo "$UNLOCKABLE_NODE" | jq -r '.name')
RESEARCH_NODE_ID=$(echo "$UNLOCKABLE_NODE" | jq -r '.id')
echo "Unlocking: $NODE_NAME ($RESEARCH_NODE_ID)"
```

Unlock it:
```bash
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/research/$RESEARCH_NODE_ID/unlock \
  -H "Authorization: Bearer $TOKEN" | jq '{unlockedNodes: .unlockedNodes}'
```

**What to verify:**
- Node appears in `unlockedNodes`
- Coins and RP decreased by the node's cost
- Subsequent GET on research shows `"isUnlocked": true` for that node

### Error case — unlock already-unlocked node
```bash
curl -s -X POST \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/research/$RESEARCH_NODE_ID/unlock \
  -H "Authorization: Bearer $TOKEN" | jq
```
Expected: HTTP 409, `"code": "conflict"`.

---

## Step 15 — Pause and Resume a Building

Pause the extractor:
```bash
curl -s -X PATCH \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$EXTRACTOR_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"isActive":false}' | jq '.isActive'
```
Expected: `false`

Run a tick — verify the extractor's output slot does not increase:
```bash
BEFORE=$(curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$EXTRACTOR_ID \
  -H "Authorization: Bearer $TOKEN" | jq '.outputSlots')

curl -s -X POST http://localhost:8080/api/v1/worlds/$WORLD_ID/tick \
  -H "Authorization: Bearer $TOKEN" > /dev/null

AFTER=$(curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$EXTRACTOR_ID \
  -H "Authorization: Bearer $TOKEN" | jq '.outputSlots')

echo "Before: $BEFORE"
echo "After:  $AFTER"
```

Resume:
```bash
curl -s -X PATCH \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$EXTRACTOR_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"isActive":true}' | jq '.isActive'
```

---

## Step 16 — Debug Map Visualisation

If `DEBUG_ROUTES=true` (default in `.env`):

```bash
# Get the SVG (requires token in Authorization header)
curl -s http://localhost:8080/api/v1/worlds/$WORLD_ID/debug/map.svg \
  -H "Authorization: Bearer $TOKEN" -o /tmp/map.svg

open /tmp/map.svg   # macOS — opens in browser
```

Or open the interactive HTML viewer directly:
```
http://localhost:8080/debug/map/$WORLD_ID?token=$TOKEN
```

This renders resource nodes, buildings, and connections on the world grid.

---

## Step 17 — Delete a Building

Disconnect the conveyor before deleting it:
```bash
# Remove extractor → conveyor connection
curl -s -X DELETE \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$EXTRACTOR_ID/connect/$CONVEYOR_ID \
  -H "Authorization: Bearer $TOKEN"

# Delete the conveyor
curl -s -X DELETE \
  http://localhost:8080/api/v1/worlds/$WORLD_ID/buildings/$CONVEYOR_ID \
  -H "Authorization: Bearer $TOKEN"
echo "Delete status: $?"
```
Expected: HTTP 204.

---

## Step 18 — Unit Tests

Run the simulation engine's unit test suite (no Docker or emulators needed):

```bash
go test ./internal/simulation/... -v
```

These tests cover:
- Extractor production rates and tier multipliers
- Conveyor belt chains and topological sort
- Splitter round-robin distribution
- Merger aggregation
- Factory recipe processing
- Research lab coin/RP generation
- Offline simulation (large tick deltas)
- Cycle detection

---

## Step 19 — Delete the World

Clean up when done:
```bash
curl -s -X DELETE http://localhost:8080/api/v1/worlds/$WORLD_ID \
  -H "Authorization: Bearer $TOKEN"
echo "Delete world status: $?"
```
Expected: HTTP 204. Verify the world and all sub-collections are removed in the Firestore Emulator UI.

---

## Quick Reference — All Variables Used

```bash
# Set these at the start of a test session:
TOKEN=<id-token from Step 2>
WORLD_ID=<world id from Step 5>
NODE_ID=<iron mine node id from Step 6>
NODE_X=<x from Step 6>
NODE_Y=<y from Step 6>
EXTRACTOR_ID=<extractor id from Step 7>
CONVEYOR_ID=<conveyor id from Step 8>
SMELTER_ID=<smelter id from Step 8>
LAB_ID=<research lab id from Step 12>
```
