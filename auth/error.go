package auth

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidAccessToken = errors.New("invalid access token")
	ErrInvalidUserData = errors.New("invalid user data")
)