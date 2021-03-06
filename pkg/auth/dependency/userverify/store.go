package userverify

import (
	"database/sql"

	"github.com/skygeario/skygear-server/pkg/core/db"
)

type Store interface {
	CreateVerifyCode(code *VerifyCode) error
	MarkConsumed(codeID string) error
	GetVerifyCodeByUser(userID string) (*VerifyCode, error)
}

type storeImpl struct {
	sqlBuilder  db.SQLBuilder
	sqlExecutor db.SQLExecutor
}

func NewStore(builder db.SQLBuilder, executor db.SQLExecutor) Store {
	return &storeImpl{
		sqlBuilder:  builder,
		sqlExecutor: executor,
	}
}

func (s *storeImpl) CreateVerifyCode(code *VerifyCode) (err error) {
	builder := s.sqlBuilder.Tenant().
		Insert(s.sqlBuilder.FullTableName("verify_code")).
		Columns(
			"id",
			"user_id",
			"login_id_key",
			"login_id",
			"code",
			"consumed",
			"created_at",
		).
		Values(
			code.ID,
			code.UserID,
			code.LoginIDKey,
			code.LoginID,
			code.Code,
			code.Consumed,
			code.CreatedAt,
		)

	_, err = s.sqlExecutor.ExecWith(builder)
	return
}

func (s *storeImpl) MarkConsumed(codeID string) (err error) {
	builder := s.sqlBuilder.Tenant().
		Update(s.sqlBuilder.FullTableName("verify_code")).
		Set("consumed", true).
		Where("id = ?", codeID)

	if _, err = s.sqlExecutor.ExecWith(builder); err != nil {
		return err
	}

	return
}

func (s *storeImpl) GetVerifyCodeByUser(userID string) (*VerifyCode, error) {
	builder := s.sqlBuilder.Tenant().
		Select(
			"id",
			"code",
			"user_id",
			"login_id_key",
			"login_id",
			"consumed",
			"created_at",
		).
		From(s.sqlBuilder.FullTableName("verify_code")).
		Where("user_id = ?", userID).
		OrderBy("created_at desc")
	scanner, err := s.sqlExecutor.QueryRowWith(builder)
	if err != nil {
		return nil, err
	}

	verifyCode := VerifyCode{}
	err = scanner.Scan(
		&verifyCode.ID,
		&verifyCode.Code,
		&verifyCode.UserID,
		&verifyCode.LoginIDKey,
		&verifyCode.LoginID,
		&verifyCode.Consumed,
		&verifyCode.CreatedAt,
	)
	if err == sql.ErrNoRows {
		err = ErrCodeNotFound
	}

	return &verifyCode, err
}

var _ Store = &storeImpl{}
