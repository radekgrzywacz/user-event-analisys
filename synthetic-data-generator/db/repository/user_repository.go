package repository

import (
	"database/sql"
	"fmt"
	"synthetic-data-generator/entity"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(user *entity.User) error {
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
