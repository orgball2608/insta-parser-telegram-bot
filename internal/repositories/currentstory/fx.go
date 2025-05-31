package currentstory

import (
	"go.uber.org/fx"
)

var Module = fx.Provide(
	fx.Annotate(
		NewPgxRepository,
		fx.As(new(Repository)),
	),
)
