package firestore

import (
	"context"
	"log/slog"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// New returns a Firestore client using Application Default Credentials.
// In local dev, set GOOGLE_APPLICATION_CREDENTIALS or run `gcloud auth application-default login`.
// On Cloud Run, ADC resolves automatically via the attached service account.
// If credsPath is non-empty, a service account key file is used instead (useful for CI).
func New(ctx context.Context, projectID, credsPath string) (*firestore.Client, error) {
	var opts []option.ClientOption
	if credsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credsPath))
		slog.Info("firestore: using service account key", "path", credsPath)
	} else {
		slog.Info("firestore: using Application Default Credentials")
	}
	return firestore.NewClient(ctx, projectID, opts...)
}
