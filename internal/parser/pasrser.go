package parser

type Client interface {
	ParseStories(username string) error
}
