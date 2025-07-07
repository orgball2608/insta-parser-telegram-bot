package unitofwork

import "go.uber.org/fx"

var Module = fx.Provide(
	NewUnitOfWork,
)
