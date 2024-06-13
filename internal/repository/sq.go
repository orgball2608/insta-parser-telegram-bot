package repository

import (
	"errors"

	"github.com/Masterminds/squirrel"
)

var SqBuilder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

var ErrBadQuery = errors.New("bad query")
