package user

import (
	"fmt"

	"github.com/brianvoe/gofakeit/v7"
)

type UserService struct {
	faker *gofakeit.Faker
	store Store
}

func NewUserService(faker *gofakeit.Faker, store Store) *UserService {
	return &UserService{
		faker: faker,
		store: store,
	}
}

func (s *UserService) GetAvailableUsers(neededCount int) ([]User, error) {
	return s.store.GetAvailableUsers(neededCount)
}

func (s *UserService) CreateNew() (User, error) {
	user := User{
		Username:  s.faker.Username(),
		Country:   s.faker.Country(),
		IP:        s.faker.IPv4Address(),
		UserAgent: s.faker.UserAgent(),
	}

	err := s.store.Save(user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *UserService) CreateMore(amount int) ([]User, error) {
	users := make([]User, 0, amount)
	for range amount {
		user, err := s.CreateNew()
		if err != nil {
			return nil, fmt.Errorf("Error creating more users: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *UserService) GetUserById(userId int) (User, error) {
	user, err := s.store.GetUserById(userId)
	if err != nil {
		return User{}, fmt.Errorf("Error retrieving user: %w", err)
	}
	return user, nil
}
