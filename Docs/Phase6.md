# Launch Docker Desktop

1. Build the image

> docker build -t factorycraftbuilder .

2. Set up your env file

>cp .env.example .env

# Edit .env with your actual values (GCP project ID, Firebase creds path, etc.)

3. Run the container

>docker run -p 8080:8080 \
  --env-file .env \
  -v /path/to/your/serviceAccountKey.json:/serviceAccountKey.json \
  factorycraftbuilder

The server will be available at http://localhost:8080.

### Notes:

The Firebase service account key needs to be mounted into the container since it's on your host machine. Adjust the path to match where your serviceAccountKey.json lives.

If FIREBASE_CREDS_PATH in your .env is ./serviceAccountKey.json, inside the container that resolves to /serviceAccountKey.json — so mount it there.

Alternatively, you can use Application Default Credentials and leave FIREBASE_CREDS_PATH empty, then mount your GCP credentials instead.
Optional: Docker Desktop GUI

After building, the image appears in Docker Desktop under Images. Click Run, expand Optional settings, set port 8080 → 8080, add your env vars, and start it from there.

