// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package session

import (
	"github.com/skygeario/skygear-server/pkg/auth"
	auth2 "github.com/skygeario/skygear-server/pkg/auth/dependency/auth"
	redis2 "github.com/skygeario/skygear-server/pkg/auth/dependency/auth/redis"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/session"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/session/redis"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo/pq"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/time"
	"net/http"
)

// Injectors from wire.go:

func newResolveHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	insecureCookieConfig := auth.ProvideSessionInsecureCookieConfig(m)
	context := auth.ProvideContext(r)
	tenantConfiguration := auth.ProvideTenantConfig(context)
	cookieConfiguration := session.ProvideSessionCookieConfiguration(r, insecureCookieConfig, tenantConfiguration)
	provider := time.NewProvider()
	requestID := auth.ProvideLoggingRequestID(r)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	store := redis.ProvideStore(context, tenantConfiguration, provider, factory)
	eventStore := redis2.ProvideEventStore(context, tenantConfiguration)
	accessEventProvider := auth2.AccessEventProvider{
		Store: eventStore,
	}
	sessionProvider := session.ProvideSessionProvider(r, store, accessEventProvider, tenantConfiguration)
	resolver := session.ProvideSessionResolver(cookieConfiguration, sessionProvider)
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	authinfoStore := pq.ProvideStore(sqlBuilderFactory, sqlExecutor)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	middleware := &auth2.Middleware{
		IDPSessionResolver: resolver,
		AccessEvents:       accessEventProvider,
		AuthInfoStore:      authinfoStore,
		TxContext:          txContext,
		Time:               provider,
	}
	handler := provideResolveHandler(middleware, provider)
	return handler
}

// wire.go:

func provideResolveHandler(m *auth2.Middleware, t time.Provider) http.Handler {
	return m.Handle(&ResolveHandler{
		TimeProvider: t,
	})
}
