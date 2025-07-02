package enidpsync

import (
	"context"

	"github.com/golang-jwt/jwt/v4"

	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/idpsync"
	"github.com/coder/coder/v2/codersdk"
)

func (e EnterpriseIDPSync) OrganizationSyncEntitled() bool {
	return true
}

func (e EnterpriseIDPSync) OrganizationSyncEnabled(ctx context.Context, db database.Store) bool {
	settings, err := e.OrganizationSyncSettings(ctx, db)
	if err == nil && settings.Field != "" {
		return true
	}
	return false
}

func (e EnterpriseIDPSync) ParseOrganizationClaims(ctx context.Context, mergedClaims jwt.MapClaims) (idpsync.OrganizationParams, *idpsync.HTTPError) {
	return idpsync.OrganizationParams{
		SyncEntitled: true,
		MergedClaims: mergedClaims,
	}, nil
}
