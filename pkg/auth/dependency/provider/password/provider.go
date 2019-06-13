package password

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/passwordhistory"
	pqPWHistory "github.com/skygeario/skygear-server/pkg/auth/dependency/passwordhistory/pq"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/skydb"
)

var (
	timeNow = func() time.Time { return time.Now().UTC() }
)

type providerImpl struct {
	sqlBuilder             db.SQLBuilder
	sqlExecutor            db.SQLExecutor
	logger                 *logrus.Entry
	loginIDsKeyWhitelist   []string
	loginIDChecker         loginIDChecker
	passwordHistoryEnabled bool
	passwordHistoryStore   passwordhistory.Store
}

func newProvider(
	builder db.SQLBuilder,
	executor db.SQLExecutor,
	logger *logrus.Entry,
	loginIDsKeyWhitelist []string,
	passwordHistoryEnabled bool,
) *providerImpl {
	return &providerImpl{
		sqlBuilder:           builder,
		sqlExecutor:          executor,
		logger:               logger,
		loginIDsKeyWhitelist: loginIDsKeyWhitelist,
		loginIDChecker: defaultLoginIDChecker{
			loginIDsKeyWhitelist: loginIDsKeyWhitelist,
		},
		passwordHistoryEnabled: passwordHistoryEnabled,
		passwordHistoryStore: pqPWHistory.NewPasswordHistoryStore(
			builder, executor, logger,
		),
	}
}

func NewProvider(
	builder db.SQLBuilder,
	executor db.SQLExecutor,
	logger *logrus.Entry,
	loginIDsKeyWhitelist []string,
	passwordHistoryEnabled bool,
) Provider {
	return newProvider(builder, executor, logger, loginIDsKeyWhitelist, passwordHistoryEnabled)
}

func (p providerImpl) IsLoginIDValid(loginID map[string]string) bool {
	return p.loginIDChecker.isValid(loginID)
}

func (p providerImpl) CreatePrincipalsByLoginID(authInfoID string, password string, loginID map[string]string) (err error) {
	for k, v := range loginID {
		principal := NewPrincipal()
		principal.UserID = authInfoID
		principal.LoginIDKey = k
		principal.LoginID = v
		principal.PlainPassword = password
		err = p.CreatePrincipal(principal)

		if err != nil {
			return
		}
	}

	return
}

func (p providerImpl) CreatePrincipal(principal Principal) (err error) {
	// TODO: log

	// Create principal
	builder := p.sqlBuilder.Insert(p.sqlBuilder.FullTableName("principal")).Columns(
		"id",
		"provider",
		"user_id",
	).Values(
		principal.ID,
		providerPassword,
		principal.UserID,
	)

	_, err = p.sqlExecutor.ExecWith(builder)
	if err != nil {
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(principal.PlainPassword), bcrypt.DefaultCost)
	if err != nil {
		panic("provider_password: Failed to hash password")
	}

	builder = p.sqlBuilder.Insert(p.sqlBuilder.FullTableName("provider_password")).Columns(
		"principal_id",
		"login_id_key",
		"login_id",
		"password",
	).Values(
		principal.ID,
		principal.LoginIDKey,
		principal.LoginID,
		hashedPassword,
	)

	_, err = p.sqlExecutor.ExecWith(builder)
	if err != nil {
		if db.IsUniqueViolated(err) {
			err = skydb.ErrUserDuplicated
		}
	}

	if p.passwordHistoryEnabled {
		p.passwordHistoryStore.CreatePasswordHistory(
			principal.UserID, hashedPassword, timeNow(),
		)
	}

	return
}

func (p providerImpl) GetPrincipalByLoginID(loginIDKey string, loginID string, principal *Principal) (err error) {
	builder := p.sqlBuilder.Select("principal_id", "password").
		From(p.sqlBuilder.FullTableName("provider_password")).
		Where(`login_id_key = ? AND login_id = ?`, loginIDKey, loginID)
	scanner := p.sqlExecutor.QueryRowWith(builder)

	err = scanner.Scan(
		&principal.ID,
		&principal.HashedPassword,
	)

	if err == sql.ErrNoRows {
		err = skydb.ErrUserNotFound
	}

	if err != nil {
		return
	}

	principal.LoginIDKey = loginIDKey
	principal.LoginID = loginID

	builder = p.sqlBuilder.Select("user_id").
		From(p.sqlBuilder.FullTableName("principal")).
		Where("id = ? AND provider = 'password'", principal.ID)
	scanner = p.sqlExecutor.QueryRowWith(builder)
	err = scanner.Scan(&principal.UserID)

	if err == sql.ErrNoRows {
		p.logger.Warnf("Missing principal for provider_password: %v", principal.ID)
		err = skydb.ErrUserNotFound
	}

	if err != nil {
		return err
	}

	return
}

func (p providerImpl) GetPrincipalsByUserID(userID string) (principals []*Principal, err error) {
	builder := p.sqlBuilder.Select("id", "user_id").
		From(p.sqlBuilder.FullTableName("principal")).
		Where("user_id = ? AND provider = 'password'", userID)
	rows, err := p.sqlExecutor.QueryWith(builder)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var principal Principal
		if err = rows.Scan(
			&principal.ID,
			&principal.UserID,
		); err != nil {
			return nil, err
		}

		principals = append(principals, &principal)
	}

	if len(principals) == 0 {
		err = skydb.ErrUserNotFound
		return
	}

	for _, principal := range principals {
		builder = p.sqlBuilder.Select("login_id_key", "login_id", "password").
			From(p.sqlBuilder.FullTableName("provider_password")).
			Where(`principal_id = ?`, principal.ID)
		scanner := p.sqlExecutor.QueryRowWith(builder)
		err = scanner.Scan(
			&principal.LoginIDKey,
			&principal.LoginID,
			&principal.HashedPassword,
		)

		if err == sql.ErrNoRows {
			err = skydb.ErrUserNotFound
		}

		if err != nil {
			return
		}
	}

	return
}

func (p providerImpl) GetPrincipalsByEmail(email string) (principals []*Principal, err error) {
	builder := p.sqlBuilder.Select("principal_id", "password").
		From(p.sqlBuilder.FullTableName("provider_password")).
		Where(`login_id_key = ? AND login_id = ?`, "email", email)
	rows, err := p.sqlExecutor.QueryWith(builder)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var principal Principal
		principal.LoginIDKey = "email"
		principal.LoginID = email
		if err = rows.Scan(
			&principal.ID,
			&principal.HashedPassword,
		); err != nil {
			return
		}

		principals = append(principals, &principal)
	}

	if len(principals) == 0 {
		err = skydb.ErrUserNotFound
		return
	}

	for _, principal := range principals {
		builder = p.sqlBuilder.Select("user_id").
			From(p.sqlBuilder.FullTableName("principal")).
			Where("id = ? AND provider = 'password'", principal.ID)
		scanner := p.sqlExecutor.QueryRowWith(builder)
		err = scanner.Scan(&principal.UserID)

		if err == sql.ErrNoRows {
			p.logger.Warnf("Missing principal for provider_password: %v", principal.ID)
			err = skydb.ErrUserNotFound
		}
		if err != nil {
			return nil, err
		}
	}

	return
}

func (p providerImpl) UpdatePrincipal(principal Principal) (err error) {
	// TODO: log

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(principal.PlainPassword), bcrypt.DefaultCost)
	if err != nil {
		panic("provider_password: Failed to hash password")
	}

	builder := p.sqlBuilder.Update(p.sqlBuilder.FullTableName("provider_password")).
		Set("login_id_key", principal.LoginIDKey).
		Set("login_id", principal.LoginID).
		Set("password", hashedPassword).
		Where("principal_id = ?", principal.ID)

	_, err = p.sqlExecutor.ExecWith(builder)
	if err != nil {
		if db.IsUniqueViolated(err) {
			err = skydb.ErrUserDuplicated
		}

		return
	}

	var isPasswordChanged = !principal.IsSamePassword(principal.PlainPassword)
	principal.HashedPassword = hashedPassword

	if p.passwordHistoryEnabled && isPasswordChanged {
		err = p.passwordHistoryStore.CreatePasswordHistory(
			principal.UserID, hashedPassword, timeNow(),
		)
	}

	return
}

// this ensures that our structure conform to certain interfaces.
var (
	_ Provider = &providerImpl{}
)