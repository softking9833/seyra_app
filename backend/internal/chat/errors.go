package chat

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrNotFound          = errors.New("not found")
	ErrForbidden         = errors.New("forbidden")
	ErrCannotMessageSelf = errors.New("cannot message self")
	ErrAlreadyMember     = errors.New("already a member")
	ErrOwnerProtected    = errors.New("owner is protected")
)
