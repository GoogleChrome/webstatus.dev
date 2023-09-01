package gh

import (
	"github.com/google/go-github/v55/github"
)

type Client struct {
	client *github.Client
}

func NewClient(token string) *Client {
	ghClient := github.NewClient(nil)
	if token != "" {
		ghClient = ghClient.WithAuthToken(token)
	}
	c := &Client{
		client: ghClient,
	}

	return c
}
