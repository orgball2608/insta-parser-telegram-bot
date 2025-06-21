package post

import (
	"go.uber.org/fx"
)

var Module = fx.Module("post_repository",
	fx.Provide(
		NewPgx,
		fx.Annotate(
			func(repo *Pgx) Repository {
				return repo
			},
			fx.As(new(Repository)),
		),
	),
)
