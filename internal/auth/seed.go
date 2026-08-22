package auth

import (
	"context"
	"log"
)

// SeedDevKeys creates default API keys for development.
//
// When keys already exist, we still print their tokens in the server log so a
// fresh deployment can recover the dashboard credentials without having to
// inspect Redis directly. This is intentionally development/demo behavior;
// production deployments should disable SEED_DEMO and use managed secrets.
func SeedDevKeys(ctx context.Context, store Store) error {
	allKeys, err := store.ListAllKeys(ctx)
	if err != nil {
		return err
	}

	if len(allKeys) > 0 {
		log.Printf("Auth keys already exist (%d keys), skipping seed", len(allKeys))
		for _, key := range allKeys {
			log.Printf("Existing %s key: %s (owner: %s, name: %s)", key.Type, key.Key, key.OwnerID, key.Name)
		}
		log.Println("WARNING: development API keys are printed above for dashboard/demo access")
		return nil
	}

	log.Println("Seeding development API keys...")

	clientKey, err := GenerateAPIKey("Dev Client", "dev-client", KeyTypeClient)
	if err != nil {
		return err
	}
	if err := store.CreateKey(ctx, clientKey); err != nil {
		return err
	}
	log.Printf("Created client key: %s (owner: %s)", clientKey.Key, clientKey.OwnerID)

	workerKey, err := GenerateAPIKey("Dev Worker", "dev-worker", KeyTypeWorker)
	if err != nil {
		return err
	}
	if err := store.CreateKey(ctx, workerKey); err != nil {
		return err
	}
	log.Printf("Created worker key: %s (owner: %s)", workerKey.Key, workerKey.OwnerID)

	adminKey, err := GenerateAPIKey("Dev Admin", "dev-admin", KeyTypeAdmin)
	if err != nil {
		return err
	}
	if err := store.CreateKey(ctx, adminKey); err != nil {
		return err
	}
	log.Printf("Created admin key: %s (owner: %s)", adminKey.Key, adminKey.OwnerID)

	log.Println("Development keys seeded successfully")
	log.Println("WARNING: These are development keys. Rotate them in production!")

	return nil
}
