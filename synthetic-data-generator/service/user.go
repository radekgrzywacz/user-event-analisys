package model

import (
	"fmt"
	"synthetic-data-generator/db/repository"
	"synthetic-data-generator/entity"

	"github.com/brianvoe/gofakeit/v7"
)

type UserService struct {
	faker    gofakeit.Faker
	userRepo repository.UserRepository
}

func (s *UserService) CreateNew() (entity.User, error) {
	user := entity.User{
		Username:  s.faker.Username(),
		Country:   s.faker.Country(),
		IP:        s.faker.IPv4Address(),
		UserAgent: s.faker.UserAgent(),
	}

	err := s.userRepo.Save(&user)
	if err != nil {
		return entity.User{}, err
	}

	return user, nil
}

func (s *UserService) CreateMore(amount int) error {
	for range amount {
		_, err := s.CreateNew()
		if err != nil {
			return fmt.Errorf("Error creating more users: %w", err)
		}
	}

	return nil
}
