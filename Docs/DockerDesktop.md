# Running FactoryCraftBuilder with Docker Desktop

This guide covers running the full local development environment using Docker Desktop, including the Firebase emulators for Firestore and Auth.

---

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed and running
- Git (to clone the repo)

No Go installation or GCP account is required for local development.

---

## 1. First-Time Setup

### 1.1 Clone the repository
```bash
git clone https://github.com/smguijt/factorycraftbuilder.git
cd FactoryCraftBuilder
```

### 1.2 Create your environment file
```bash
cp .env.example .env
```

Open `.env` and set the following. For local emulator use, most values can stay as shown:

```env
PORT=8080
GCP_PROJECT_ID=demo-local        # Must start with "demo-" for emulators
FIREBASE_CREDS_PATH=              # Leave empty — emulators don't need credentials
STARTING_COINS=500
MAX_OFFLINE_SECONDS=28800
DEBUG_ROUTES=true
```

> **Note:** The `demo-` prefix in `GCP_PROJECT_ID` is required. It tells the Firebase SDK to connect to the local emulator instead of real GCP services, and no service account key is needed.

---

## 2. Starting the Stack

### 2.1 Build and start all services
```bash
docker compose up --build
```

This starts two containers:

| Container   | Purpose                          | Ports                          |
|-------------|----------------------------------|--------------------------------|
| `emulators` | Firebase Auth + Firestore        | 9099 (Auth), 8081 (Firestore)  |
| `server`    | FactoryCraftBuilder API          | 8080 (HTTP)                    |

The `server` container waits for the emulators to be healthy before starting.

### 2.2 Verify the stack is running

Check the API health endpoint:
```bash
curl http://localhost:8080/health
```
Expected response:
```json
{"status":"ok"}
```

Check the Emulator UI (Firestore data browser):
```
http://localhost:4000
```

---

## 3. Docker Desktop GUI (Alternative)

If you prefer the Docker Desktop interface over the terminal:

1. Open Docker Desktop
2. Go to **Images** → click **Run** on `factorycraftbuilder-server` (after building once)
3. Expand **Optional settings**:
   - Set **Host port** to `8080`
   - Add environment variables from your `.env` file
4. Click **Run**

For the full stack with emulators, the CLI (`docker compose up`) is easier.

---

## 4. Useful Commands

| Action                              | Command                                  |
|-------------------------------------|------------------------------------------|
| Start stack (foreground)            | `docker compose up --build`              |
| Start stack (background)            | `docker compose up --build -d`           |
| Stop stack                          | `docker compose down`                    |
| Stop and delete volumes             | `docker compose down -v`                 |
| View logs (all)                     | `docker compose logs -f`                 |
| View server logs only               | `docker compose logs -f server`          |
| View emulator logs only             | `docker compose logs -f emulators`       |
| Rebuild server only (after changes) | `docker compose build server && docker compose up -d server` |
| Open shell in server container      | `docker compose exec server sh`          |

---

## 5. Services and Ports

| Service            | URL                              | Notes                         |
|--------------------|----------------------------------|-------------------------------|
| API server         | http://localhost:8080            | Main REST API                 |
| Emulator UI        | http://localhost:4000            | Firestore browser + Auth mgmt |
| Firestore emulator | localhost:8081                   | gRPC (used by server)         |
| Auth emulator      | localhost:9099                   | REST (used by server)         |

---

## 6. Working with the Firebase Emulator UI

Open http://localhost:4000 to:

- **Firestore tab** — Browse and edit documents in real time (`players`, `worlds`, `buildings`, etc.)
- **Authentication tab** — Create test users, copy their UIDs, manage sign-in methods

### Creating a test user for API calls

1. Open http://localhost:4000 → **Authentication**
2. Click **Add user**
3. Enter an email (e.g. `test@example.com`) and any password
4. Copy the generated **User UID** — you will need this to construct auth tokens

To obtain an ID token for API testing, use the Firebase Auth REST API pointed at the emulator:

```bash
curl -s -X POST \
  "http://localhost:9099/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=fake-api-key" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"yourpassword","returnSecureToken":true}' \
  | grep -o '"idToken":"[^"]*"'
```

The `idToken` value is your Bearer token for API requests.

---

## 7. Stopping and Resetting

### Stop the stack (preserve data)
```bash
docker compose down
```

Restart with `docker compose up` — emulator data is kept in the container while it exists.

### Full reset (wipe all emulator data)
```bash
docker compose down -v
docker compose up --build
```

---

## 8. Troubleshooting

### Port already in use
```
Error: port 8080 is already allocated
```
Stop any process using that port, or change the host port in `docker-compose.yml`:
```yaml
ports:
  - "8090:8080"   # Change host port to 8090
```

### Server starts before emulators are ready
The `depends_on: condition: service_healthy` in `docker-compose.yml` handles this. If the health check times out, increase the `retries` value or check emulator logs:
```bash
docker compose logs emulators
```

### `firebase: command not found` in emulator container
Make sure you are using the `gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators` image (already set in `docker-compose.yml`). Do not replace it with a plain `gcloud` image.

### Changes to Go code not reflected
Rebuild the server image:
```bash
docker compose build server
docker compose up -d server
```
