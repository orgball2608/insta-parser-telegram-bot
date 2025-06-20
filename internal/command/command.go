package command

import "context"

type Client interface {
	HandleCommand(ctx context.Context) error
}
