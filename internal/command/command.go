package command

type Client interface {
	HandleCommand() error
}
