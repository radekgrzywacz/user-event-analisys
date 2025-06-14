package user

import (
	"database/sql"
	"fmt"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(user User) error {
	query := `
		INSERT INTO users (username, country, ip, user_agent)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	err := r.db.QueryRow(query, user.Username, user.Country, user.IP, user.UserAgent).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetAvailableUsers(neededCount int) ([]User, error) {
	query := `
		SELECT * FROM users LIMIT $1
	`

	rows, err := r.db.Query(query, neededCount)
	if err != nil {
		return nil, fmt.Errorf("Error fetching available users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Country, &user.IP, &user.UserAgent)
		if err != nil {
			return nil, fmt.Errorf("Error scanning user row: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Row iteration error: %w", err)
	}

	return users, nil
}

func (r *UserRepository) GetUserById(userId int) (User, error) {
	query := "SELECT * FROM users WHERE id = $1"

	var user User
	err := r.db.QueryRow(query, userId).Scan(&user.ID, &user.Username, &user.Country, &user.IP, &user.UserAgent)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, fmt.Errorf("user with id %d not found", userId)
		}
		return User{}, fmt.Errorf("could not retrieve user with id %d: %w", userId, err)
	}

	return user, nil
}
