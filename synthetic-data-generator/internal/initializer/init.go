package initializer

import (
	"database/sql"
	"fmt"
	"synthetic-data-generator/internal/user"

	"github.com/brianvoe/gofakeit/v7"
)

func CreateUsersIfNeeded(usersCount int, db *sql.DB, faker *gofakeit.Faker) ([]user.User, error) {
	userRepo := user.NewUserRepository(db)
	userService := user.NewUserService(faker, userRepo)

	availableUsers, err := userRepo.GetAvailableUsers(usersCount)
	if err != nil {
		return nil, fmt.Errorf("Error fetching available users")
	}
	availableUsersCount := len(availableUsers)

	if availableUsersCount < usersCount {
		difference := usersCount - availableUsersCount
		newUsers, err := userService.CreateMore(difference)
		if err != nil {
			return nil, fmt.Errorf("Error creating more users: %w", err)
		}
		availableUsers = append(availableUsers, newUsers...)
	}

	return availableUsers, nil
}
