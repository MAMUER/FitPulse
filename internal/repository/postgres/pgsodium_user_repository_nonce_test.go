package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/domain/entity"
)

func TestPgsodiumUserRepository_GenerateNonceFailures(t *testing.T) {
	orig := db.GenerateNonce
	defer func() { db.GenerateNonce = orig }()
	db.GenerateNonce = func() ([]byte, error) { return nil, errors.New("boom") }

	repo, _ := setupPgsodiumUserRepo(t)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		err := repo.Create(ctx, &entity.User{ID: "u", Email: "e@e.com", FullName: "n", PasswordHash: "h", Role: "client"})
		require.Error(t, err)
	})
	t.Run("CreateWithInvite", func(t *testing.T) {
		err := repo.CreateWithInvite(ctx, "u", "e@e.com", "hash", "h", "n", "client")
		require.Error(t, err)
	})
	t.Run("Update", func(t *testing.T) {
		err := repo.Update(ctx, &entity.User{ID: "u", FullName: "n"})
		require.Error(t, err)
	})
	t.Run("UpdateUserDetails", func(t *testing.T) {
		err := repo.UpdateUserDetails(ctx, "u", "n", "nick")
		require.Error(t, err)
	})
	t.Run("CreateYandexUser", func(t *testing.T) {
		_, _, _, err := repo.CreateYandexUser(ctx, "yid", "hash", "e@e.com", "n", "nick", "nh")
		require.Error(t, err)
	})
}
