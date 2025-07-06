package subscription

import (
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Provide(
		fx.Annotate(
			NewPgxRepository,
			fx.As(new(Repository)),
		),
	)
}
