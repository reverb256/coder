package coderd

import (
	"bytes"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/imulab/go-scim/pkg/v2/handlerutil"
	scimjson "github.com/imulab/go-scim/pkg/v2/json"
	"github.com/imulab/go-scim/pkg/v2/service"
	"github.com/imulab/go-scim/pkg/v2/spec"
	"golang.org/x/xerrors"

	"github.com/coder/coder/v2/coderd/audit"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
	"github.com/coder/coder/v2/coderd/database/dbtime"
	"github.com/coder/coder/v2/coderd/httpapi"
	"github.com/coder/coder/v2/codersdk"
)

func (api *API) scimVerifyAuthHeader(r *http.Request) bool {
	bearer := []byte("bearer ")
	hdr := []byte(r.Header.Get("Authorization"))

	// Use toLower to make the comparison case-insensitive.
	if len(hdr) >= len(bearer) && subtle.ConstantTimeCompare(bytes.ToLower(hdr[:len(bearer)]), bearer) == 1 {
		hdr = hdr[len(bearer):]
	}

	return len(api.SCIMAPIKey) != 0 && subtle.ConstantTimeCompare(hdr, api.SCIMAPIKey) == 1
}

func scimUnauthorized(rw http.ResponseWriter) {
	_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusUnauthorized, "invalidAuthorization", xerrors.New("invalid authorization")))
}

// ServiceProviderConfig returns a static SCIM service provider configuration.
func (api *API) scimServiceProviderConfig(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Content-Type", spec.ApplicationScimJson)
	rw.WriteHeader(http.StatusOK)

	providerUpdated := time.Date(2024, 10, 25, 17, 0, 0, 0, time.UTC)
	var location string
	locURL, err := api.AccessURL.Parse("/scim/v2/ServiceProviderConfig")
	if err == nil {
		location = locURL.String()
	}

	enc := json.NewEncoder(rw)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(ServiceProviderConfig{
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		DocURI:  "https://coder.com/docs/admin/users/oidc-auth#scim",
		Patch: Supported{
			Supported: true,
		},
		Bulk: BulkSupported{
			Supported: false,
		},
		Filter: FilterSupported{
			Supported: false,
		},
		ChangePassword: Supported{
			Supported: false,
		},
		Sort: Supported{
			Supported: false,
		},
		ETag: Supported{
			Supported: false,
		},
		AuthSchemes: []AuthenticationScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "HTTP Header Authentication",
				Description: "Authentication scheme using the Authorization header with the shared token",
				DocURI:      "https://coder.com/docs/admin/users/oidc-auth#scim",
			},
		},
		Meta: ServiceProviderMeta{
			Created:      providerUpdated,
			LastModified: providerUpdated,
			Location:     location,
			ResourceType: "ServiceProviderConfig",
		},
	})
}

// scimGetUsers intentionally always returns no users.
func (api *API) scimGetUsers(rw http.ResponseWriter, r *http.Request) {
	if !api.scimVerifyAuthHeader(r) {
		scimUnauthorized(rw)
		return
	}

	_ = handlerutil.WriteSearchResultToResponse(rw, &service.QueryResponse{
		TotalResults: 0,
		StartIndex:   1,
		ItemsPerPage: 0,
		Resources:    []scimjson.Serializable{},
	})
}

// scimGetUser intentionally always returns an error saying the user wasn't found.
func (api *API) scimGetUser(rw http.ResponseWriter, r *http.Request) {
	if !api.scimVerifyAuthHeader(r) {
		scimUnauthorized(rw)
		return
	}

	_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusNotFound, spec.ErrNotFound.Type, xerrors.New("endpoint will always return 404")))
}

type SCIMUser struct {
	Schemas  []string `json:"schemas"`
	ID       string   `json:"id"`
	UserName string   `json:"userName"`
	Name     struct {
		GivenName  string `json:"givenName"`
		FamilyName string `json:"familyName"`
	} `json:"name"`
	Emails []struct {
		Primary bool   `json:"primary"`
		Value   string `json:"value" format:"email"`
		Type    string `json:"type"`
		Display string `json:"display"`
	} `json:"emails"`
	Active *bool         `json:"active"`
	Groups []interface{} `json:"groups"`
	Meta   struct {
		ResourceType string `json:"resourceType"`
	} `json:"meta"`
}

var SCIMAuditAdditionalFields = map[string]string{
	"automatic_actor":     "coder",
	"automatic_subsystem": "scim",
}

// scimPostUser creates a new user, or returns the existing user if it exists.
func (api *API) scimPostUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !api.scimVerifyAuthHeader(r) {
		scimUnauthorized(rw)
		return
	}

	auditor := *api.Auditor.Load()
	aReq, commitAudit := audit.InitRequest[database.User](rw, &audit.RequestParams{
		Audit:            auditor,
		Log:              api.Logger,
		Request:          r,
		Action:           database.AuditActionCreate,
		AdditionalFields: SCIMAuditAdditionalFields,
	})
	defer commitAudit()

	var sUser SCIMUser
	err := json.NewDecoder(r.Body).Decode(&sUser)
	if err != nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidRequest", err))
		return
	}

	if sUser.Active == nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidRequest", xerrors.New("active field is required")))
		return
	}

	email := ""
	for _, e := range sUser.Emails {
		if e.Primary {
			email = e.Value
			break
		}
	}

	if email == "" {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidEmail", xerrors.New("no primary email provided")))
		return
	}

	dbUser, err := api.Database.GetUserByEmailOrUsername(dbauthz.AsSystemRestricted(ctx), database.GetUserByEmailOrUsernameParams{
		Email:    email,
		Username: sUser.UserName,
	})
	if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
		_ = handlerutil.WriteError(rw, err)
		return
	}
	if err == nil {
		sUser.ID = dbUser.ID.String()
		sUser.UserName = dbUser.Username

		if *sUser.Active && dbUser.Status == database.UserStatusSuspended {
			newUser, err := api.Database.UpdateUserStatus(dbauthz.AsSystemRestricted(r.Context()), database.UpdateUserStatusParams{
				ID:        dbUser.ID,
				Status:    database.UserStatusDormant,
				UpdatedAt: dbtime.Now(),
			})
			if err != nil {
				_ = handlerutil.WriteError(rw, err)
				return
			}
			aReq.New = newUser
		} else {
			aReq.New = dbUser
		}

		aReq.Old = dbUser

		httpapi.Write(ctx, rw, http.StatusOK, sUser)
		return
	}

	usernameValid := codersdk.NameValid(sUser.UserName)
	if usernameValid != nil {
		if sUser.UserName == "" {
			sUser.UserName = email
		}
		sUser.UserName = codersdk.UsernameFrom(sUser.UserName)
	}

	organizations := []uuid.UUID{}
	orgSync, err := api.IDPSync.OrganizationSyncSettings(dbauthz.AsSystemRestricted(ctx), api.Database)
	if err != nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusInternalServerError, "internalError", xerrors.Errorf("failed to get organization sync settings: %w", err)))
		return
	}
	if orgSync.AssignDefault {
		defaultOrganization, err := api.Database.GetDefaultOrganization(dbauthz.AsSystemRestricted(ctx))
		if err != nil {
			_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusInternalServerError, "internalError", xerrors.Errorf("failed to get default organization: %w", err)))
			return
		}
		organizations = append(organizations, defaultOrganization.ID)
	}

	dbUser, err = api.CreateUser(dbauthz.AsSystemRestricted(ctx), api.Database, CreateUserRequest{
		CreateUserRequestWithOrgs: codersdk.CreateUserRequestWithOrgs{
			Username:        sUser.UserName,
			Email:           email,
			OrganizationIDs: organizations,
		},
		LoginType:        database.LoginTypeOIDC,
		SkipNotifications: true,
	})
	if err != nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusInternalServerError, "internalError", xerrors.Errorf("failed to create user: %w", err)))
		return
	}
	aReq.New = dbUser
	aReq.UserID = dbUser.ID

	sUser.ID = dbUser.ID.String()
	sUser.UserName = dbUser.Username

	httpapi.Write(ctx, rw, http.StatusOK, sUser)
}

// scimPatchUser supports suspending and activating users only.
func (api *API) scimPatchUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !api.scimVerifyAuthHeader(r) {
		scimUnauthorized(rw)
		return
	}

	auditor := *api.Auditor.Load()
	aReq, commitAudit := audit.InitRequestWithCancel[database.User](rw, &audit.RequestParams{
		Audit:   auditor,
		Log:     api.Logger,
		Request: r,
		Action:  database.AuditActionWrite,
	})

	defer commitAudit(true)

	id := chi.URLParam(r, "id")

	var sUser SCIMUser
	err := json.NewDecoder(r.Body).Decode(&sUser)
	if err != nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidRequest", err))
		return
	}
	sUser.ID = id

	uid, err := uuid.Parse(id)
	if err != nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidId", xerrors.Errorf("id must be a uuid: %w", err)))
		return
	}

	dbUser, err := api.Database.GetUserByID(dbauthz.AsSystemRestricted(ctx), uid)
	if err != nil {
		_ = handlerutil.WriteError(rw, err)
		return
	}
	aReq.Old = dbUser
	aReq.UserID = dbUser.ID

	if sUser.Active == nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidRequest", xerrors.New("active field is required")))
		return
	}

	newStatus := scimUserStatus(dbUser, *sUser.Active)
	if dbUser.Status != newStatus {
		userNew, err := api.Database.UpdateUserStatus(dbauthz.AsSystemRestricted(r.Context()), database.UpdateUserStatusParams{
			ID:        dbUser.ID,
			Status:    newStatus,
			UpdatedAt: dbtime.Now(),
		})
		if err != nil {
			_ = handlerutil.WriteError(rw, err)
			return
		}
		dbUser = userNew
	} else {
		commitAudit(false)
	}

	aReq.New = dbUser
	httpapi.Write(ctx, rw, http.StatusOK, sUser)
}

// scimPutUser supports suspending and activating users only.
func (api *API) scimPutUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !api.scimVerifyAuthHeader(r) {
		scimUnauthorized(rw)
		return
	}

	auditor := *api.Auditor.Load()
	aReq, commitAudit := audit.InitRequestWithCancel[database.User](rw, &audit.RequestParams{
		Audit:   auditor,
		Log:     api.Logger,
		Request: r,
		Action:  database.AuditActionWrite,
	})

	defer commitAudit(true)

	id := chi.URLParam(r, "id")

	var sUser SCIMUser
	err := json.NewDecoder(r.Body).Decode(&sUser)
	if err != nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidRequest", err))
		return
	}
	sUser.ID = id
	if sUser.Active == nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidRequest", xerrors.New("active field is required")))
		return
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "invalidId", xerrors.Errorf("id must be a uuid: %w", err)))
		return
	}

	dbUser, err := api.Database.GetUserByID(dbauthz.AsSystemRestricted(ctx), uid)
	if err != nil {
		_ = handlerutil.WriteError(rw, err)
		return
	}
	aReq.Old = dbUser
	aReq.UserID = dbUser.ID

	if immutabilityViolation(dbUser.Username, sUser.UserName) {
		_ = handlerutil.WriteError(rw, NewSCIMHTTPError(http.StatusBadRequest, "mutability", xerrors.Errorf("username is currently an immutable field, and cannot be changed. Current: %s, New: %s", dbUser.Username, sUser.UserName)))
		return
	}

	newStatus := scimUserStatus(dbUser, *sUser.Active)
	if dbUser.Status != newStatus {
		userNew, err := api.Database.UpdateUserStatus(dbauthz.AsSystemRestricted(r.Context()), database.UpdateUserStatusParams{
			ID:        dbUser.ID,
			Status:    newStatus,
			UpdatedAt: dbtime.Now(),
		})
		if err != nil {
			_ = handlerutil.WriteError(rw, err)
			return
		}
		dbUser = userNew
	} else {
		commitAudit(false)
	}

	aReq.New = dbUser
	httpapi.Write(ctx, rw, http.StatusOK, sUser)
}

func immutabilityViolation[T comparable](old, newVal T) bool {
	var empty T
	if newVal == empty {
		return false
	}
	return old != newVal
}

func scimUserStatus(user database.User, active bool) database.UserStatus {
	if !active {
		return database.UserStatusSuspended
	}

	switch user.Status {
	case database.UserStatusActive:
		return database.UserStatusActive
	case database.UserStatusDormant, database.UserStatusSuspended:
		return database.UserStatusDormant
	default:
		return database.UserStatusDormant
	}
}

// --- Minimal SCIM types for ServiceProviderConfig ---

type Supported struct {
	Supported bool `json:"supported"`
}

type BulkSupported struct {
	Supported bool `json:"supported"`
}

type FilterSupported struct {
	Supported bool `json:"supported"`
}

type AuthenticationScheme struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DocURI      string `json:"documentationUri"`
}

type ServiceProviderMeta struct {
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Location     string    `json:"location"`
	ResourceType string    `json:"resourceType"`
}

type ServiceProviderConfig struct {
	Schemas       []string              `json:"schemas"`
	DocURI        string                `json:"documentationUri"`
	Patch         Supported             `json:"patch"`
	Bulk          BulkSupported         `json:"bulk"`
	Filter        FilterSupported       `json:"filter"`
	ChangePassword Supported            `json:"changePassword"`
	Sort          Supported             `json:"sort"`
	ETag          Supported             `json:"etag"`
	AuthSchemes   []AuthenticationScheme `json:"authenticationSchemes"`
	Meta          ServiceProviderMeta   `json:"meta"`
}

// NewSCIMHTTPError creates a new SCIM HTTP error.
func NewSCIMHTTPError(status int, scimType string, err error) error {
	return &SCIMHTTPError{
		Status:   status,
		ScimType: scimType,
		Err:      err,
	}
}

type SCIMHTTPError struct {
	Status   int
	ScimType string
	Err      error
}

func (e *SCIMHTTPError) Error() string {
	return e.Err.Error()
}
