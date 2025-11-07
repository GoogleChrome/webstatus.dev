package gh

import (
	"context"
	"errors"
	"log/slog"
)

// ErrFieldUnexpectedlyNil is returned when a field that is expected to be non-nil is found to be nil.
// A lot of GitHub API fields are pointers, so we need to check for nil values.
var ErrFieldUnexpectedlyNil = errors.New("expected field to be non-nil")

type GitHubUser struct {
	ID       int64
	Username string
}

func (c *UserGitHubClient) GetCurrentUser(ctx context.Context) (*GitHubUser, error) {
	// An empty string for the user parameter fetches the authenticated user.
	user, _, err := c.usersClient.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	if user.ID == nil {
		slog.ErrorContext(ctx, "missing user id after get user request", "user", user)

		return nil, ErrFieldUnexpectedlyNil
	}

	if user.Login == nil {
		slog.ErrorContext(ctx, "missing user login after get user request", "user", user)

		return nil, ErrFieldUnexpectedlyNil
	}

	return &GitHubUser{
		ID:       *user.ID,
		Username: *user.Login,
	}, nil
}
