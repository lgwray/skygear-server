package sso

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/oauth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/sso"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"

	"github.com/skygeario/skygear-server/pkg/auth"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
)

func AttachLinkHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/sso/{provider}/link", &LinkHandlerFactory{
		Dependency: authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type LinkHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f LinkHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &LinkHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	vars := mux.Vars(request)
	h.ProviderID = vars["provider"]
	h.Provider = h.ProviderFactory.NewProvider(h.ProviderID)
	return h.RequireAuthz(handler.APIHandlerToHandler(hook.WrapHandler(h.HookProvider, h), h.TxContext), h)
}

// LinkRequestPayload login handler request payload
type LinkRequestPayload struct {
	AccessToken string `json:"access_token"`
}

// @JSONSchema
const LinkRequestSchema = `
{
	"$id": "#LinkRequest",
	"type": "object",
	"properties": {
		"access_token": { "type": "string" }
	}
}
`

// Validate request payload
func (p LinkRequestPayload) Validate() error {
	if p.AccessToken == "" {
		return skyerr.NewInvalidArgument("empty access token", []string{"access_token"})
	}

	return nil
}

/*
	@Operation POST /sso/{provider_id}/link - Link SSO provider with token
		Link the specified SSO provider with the current user, using access
		token obtained from the provider.

		@Tag SSO
		@SecurityRequirement access_key
		@SecurityRequirement access_token

		@Parameter {SSOProviderID}
		@RequestBody
			Describe the access token of SSO provider.
			@JSONSchema {LinkRequest}
		@Response 200 {EmptyResponse}

		@Callback identity_create {UserSyncEvent}
		@Callback user_sync {UserSyncEvent}
*/
type LinkHandler struct {
	TxContext          db.TxContext               `dependency:"TxContext"`
	AuthContext        coreAuth.ContextGetter     `dependency:"AuthContextGetter"`
	RequireAuthz       handler.RequireAuthz       `dependency:"RequireAuthz"`
	OAuthAuthProvider  oauth.Provider             `dependency:"OAuthAuthProvider"`
	IdentityProvider   principal.IdentityProvider `dependency:"IdentityProvider"`
	AuthInfoStore      authinfo.Store             `dependency:"AuthInfoStore"`
	UserProfileStore   userprofile.Store          `dependency:"UserProfileStore"`
	HookProvider       hook.Provider              `dependency:"HookProvider"`
	ProviderFactory    *sso.ProviderFactory       `dependency:"SSOProviderFactory"`
	OAuthConfiguration config.OAuthConfiguration  `dependency:"OAuthConfiguration"`
	Provider           sso.OAuthProvider
	ProviderID         string
}

func (h LinkHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		authz.PolicyFunc(policy.RequireAuthenticated),
		authz.PolicyFunc(policy.DenyDisabledUser),
	)
}

func (h LinkHandler) WithTx() bool {
	return true
}

func (h LinkHandler) DecodeRequest(request *http.Request, resp http.ResponseWriter) (handler.RequestPayload, error) {
	payload := LinkRequestPayload{}
	err := handler.DecodeJSONBody(request, resp, &payload)
	if err != nil {
		return payload, err
	}
	return payload, nil
}

func (h LinkHandler) Handle(req interface{}) (resp interface{}, err error) {
	if !h.OAuthConfiguration.ExternalAccessTokenFlowEnabled {
		err = skyerr.NewError(skyerr.UndefinedOperation, "External access token flow is disabled")
		return
	}

	provider, ok := h.Provider.(sso.ExternalAccessTokenFlowProvider)
	if !ok {
		err = skyerr.NewInvalidArgument("Provider is not supported", []string{h.ProviderID})
		return
	}

	authInfo, _ := h.AuthContext.AuthInfo()
	userID := authInfo.ID
	payload := req.(LinkRequestPayload)

	linkState := sso.LinkState{
		UserID: userID,
	}
	oauthAuthInfo, err := provider.ExternalAccessTokenGetAuthInfo(sso.NewBearerAccessTokenResp(payload.AccessToken))
	if err != nil {
		return
	}

	handler := respHandler{
		AuthInfoStore:     h.AuthInfoStore,
		OAuthAuthProvider: h.OAuthAuthProvider,
		IdentityProvider:  h.IdentityProvider,
		UserProfileStore:  h.UserProfileStore,
		HookProvider:      h.HookProvider,
	}
	resp, err = handler.linkActionResp(oauthAuthInfo, linkState)

	return
}