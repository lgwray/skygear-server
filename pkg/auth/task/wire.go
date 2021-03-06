//+build wireinject

package task

import (
	"context"

	"github.com/google/wire"

	pkg "github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/identity/loginid"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/user"
	"github.com/skygeario/skygear-server/pkg/core/async"
)

func newVerifyCodeSendTask(ctx context.Context, m pkg.DependencyMap) async.Task {
	wire.Build(
		pkg.CommonDependencySet,
		wire.Bind(new(VerifyCodeLoginIDProvider), new(*loginid.Provider)),
		wire.Bind(new(UserProvider), new(*user.Queries)),
		wire.Struct(new(VerifyCodeSendTask), "*"),
		wire.Bind(new(async.Task), new(*VerifyCodeSendTask)),
	)
	return nil
}

func newPwHouseKeeperTask(ctx context.Context, m pkg.DependencyMap) async.Task {
	wire.Build(
		pkg.CommonDependencySet,
		wire.Struct(new(PwHousekeeperTask), "*"),
		wire.Bind(new(async.Task), new(*PwHousekeeperTask)),
	)
	return nil
}

func newSendMessagesTask(ctx context.Context, m pkg.DependencyMap) async.Task {
	wire.Build(
		pkg.CommonDependencySet,
		wire.Struct(new(SendMessagesTask), "*"),
		wire.Bind(new(async.Task), new(*SendMessagesTask)),
	)
	return nil
}
