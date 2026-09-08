package chat

import (
	"context"
	"errors"

	"seyra/backend/internal/auth"
)

type AuthDirectory struct {
	Store auth.Store
}

func NewAuthDirectory(store auth.Store) AuthDirectory {
	return AuthDirectory{Store: store}
}

func (d AuthDirectory) LookupUsername(ctx context.Context, username string) (UserRef, error) {
	user, err := d.Store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			return UserRef{}, ErrNotFound
		}
		return UserRef{}, err
	}
	return UserRef{ID: user.ID, Username: user.Username}, nil
}

func (d AuthDirectory) LookupID(ctx context.Context, id string) (UserRef, error) {
	user, err := d.Store.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			return UserRef{}, ErrNotFound
		}
		return UserRef{}, err
	}
	return UserRef{ID: user.ID, Username: user.Username}, nil
}
