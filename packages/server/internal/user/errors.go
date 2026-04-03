package user

import "errors"

var (
	ErrDisplayNameRequired    = errors.New("displayName is required")
	ErrDisplayNameTooLong     = errors.New("displayName is too long")
	ErrGitHubUsernameRequired = errors.New("github username is required")
	ErrGitHubUsernameInvalid  = errors.New("invalid github username")
)
