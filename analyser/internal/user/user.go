package user

import (
	"analyser/internal/store"
	"context"
)

func EnsureUserExists(pg *store.Queries, ctx context.Context, userId int64) error {
	_, err := pg.GetUserByID(ctx, int32(userId))
	if err == nil {
		return nil
	}

	_, err = pg.InsertUser(ctx, int32(userId))
	return err
}
