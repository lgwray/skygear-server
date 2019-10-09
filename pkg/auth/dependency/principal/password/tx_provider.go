package password

import (
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/logging"
)

type safeProviderImpl struct {
	impl      *providerImpl
	txContext db.SafeTxContext
}

func NewSafeProvider(
	builder db.SQLBuilder,
	executor db.SQLExecutor,
	loggerFactory logging.Factory,
	loginIDsKeys map[string]config.LoginIDKeyConfiguration,
	allowedRealms []string,
	passwordHistoryEnabled bool,
	txContext db.SafeTxContext,
) Provider {
	return &safeProviderImpl{
		impl:      newProvider(builder, executor, loggerFactory, loginIDsKeys, allowedRealms, passwordHistoryEnabled),
		txContext: txContext,
	}
}

func (p *safeProviderImpl) ValidateLoginIDs(loginIDs []LoginID) error {
	p.txContext.EnsureTx()
	return p.impl.ValidateLoginIDs(loginIDs)
}

func (p *safeProviderImpl) CheckLoginIDKeyType(loginIDKey string, standardKey metadata.StandardKey) bool {
	p.txContext.EnsureTx()
	return p.impl.CheckLoginIDKeyType(loginIDKey, standardKey)
}

func (p safeProviderImpl) IsRealmValid(realm string) bool {
	p.txContext.EnsureTx()
	return p.impl.IsRealmValid(realm)
}

func (p *safeProviderImpl) IsDefaultAllowedRealms() bool {
	p.txContext.EnsureTx()
	return p.impl.IsDefaultAllowedRealms()
}

func (p *safeProviderImpl) CreatePrincipalsByLoginID(authInfoID string, password string, loginIDs []LoginID, realm string) ([]*Principal, error) {
	p.txContext.EnsureTx()
	return p.impl.CreatePrincipalsByLoginID(authInfoID, password, loginIDs, realm)
}

func (p *safeProviderImpl) CreatePrincipal(principal Principal) error {
	p.txContext.EnsureTx()
	return p.impl.CreatePrincipal(principal)
}

func (p *safeProviderImpl) GetPrincipalByLoginIDWithRealm(loginIDKey string, loginID string, realm string, principal *Principal) (err error) {
	p.txContext.EnsureTx()
	return p.impl.GetPrincipalByLoginIDWithRealm(loginIDKey, loginID, realm, principal)
}

func (p *safeProviderImpl) GetPrincipalsByUserID(userID string) ([]*Principal, error) {
	p.txContext.EnsureTx()
	return p.impl.GetPrincipalsByUserID(userID)
}

func (p *safeProviderImpl) GetPrincipalsByLoginID(loginIDKey string, loginID string) ([]*Principal, error) {
	p.txContext.EnsureTx()
	return p.impl.GetPrincipalsByLoginID(loginIDKey, loginID)
}

func (p *safeProviderImpl) UpdatePassword(principal *Principal, password string) error {
	p.txContext.EnsureTx()
	return p.impl.UpdatePassword(principal, password)
}

func (p *safeProviderImpl) MigratePassword(principal *Principal, password string) error {
	p.txContext.EnsureTx()
	return p.impl.MigratePassword(principal, password)
}

func (p *safeProviderImpl) ID() string {
	p.txContext.EnsureTx()
	return p.impl.ID()
}

func (p *safeProviderImpl) GetPrincipalByID(principalID string) (principal.Principal, error) {
	p.txContext.EnsureTx()
	return p.impl.GetPrincipalByID(principalID)
}

func (p *safeProviderImpl) ListPrincipalsByUserID(userID string) ([]principal.Principal, error) {
	p.txContext.EnsureTx()
	return p.impl.ListPrincipalsByUserID(userID)
}

func (p *safeProviderImpl) ListPrincipalsByClaim(claimName string, claimValue string) ([]principal.Principal, error) {
	p.txContext.EnsureTx()
	return p.impl.ListPrincipalsByClaim(claimName, claimValue)
}

var (
	_ Provider = &safeProviderImpl{}
)