package enidpsync

import (
	"context"

	"github.com/golang-jwt/jwt/v4"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/idpsync"
	"github.com/coder/coder/v2/codersdk"
)

func (e EnterpriseIDPSync) OrganizationSyncEntitled() bool {
<<<<<<< HEAD
	return true
}

func (e EnterpriseIDPSync) OrganizationSyncEnabled(ctx context.Context, db database.Store) bool {
=======
	return e.entitlements.Enabled(codersdk.FeatureMultipleOrganizations)
}

func (e EnterpriseIDPSync) OrganizationSyncEnabled(ctx context.Context, db database.Store) bool {
	if !e.OrganizationSyncEntitled() {
		return false
	}

	// If this logic is ever updated, make sure to update the corresponding
	// checkIDPOrgSync in coderd/telemetry/telemetry.go.
>>>>>>> upstream/main
	settings, err := e.OrganizationSyncSettings(ctx, db)
	if err == nil && settings.Field != "" {
		return true
	}
	return false
}

func (e EnterpriseIDPSync) ParseOrganizationClaims(ctx context.Context, mergedClaims jwt.MapClaims) (idpsync.OrganizationParams, *idpsync.HTTPError) {
<<<<<<< HEAD
	return idpsync.OrganizationParams{
=======
	if !e.OrganizationSyncEntitled() {
		// Default to agpl if multi-org is not enabled
		return e.AGPLIDPSync.ParseOrganizationClaims(ctx, mergedClaims)
	}

	return idpsync.OrganizationParams{
		// Return true if entitled
>>>>>>> upstream/main
		SyncEntitled: true,
		MergedClaims: mergedClaims,
	}, nil
}
