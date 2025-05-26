package model

import "github.com/brianvoe/gofakeit/v7"

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Country   string `json:"country"`
	IP        string `json:"ip"`
	UserAgent string `json:"user-agent"`
}

func createNew(faker *gofakeit.Faker) User {
	user := User{
		Username:  faker.Username(),
		Country:   faker.Country(),
		IP:        faker.IPv4Address(),
		UserAgent: faker.UserAgent(),
	}
	
	
}
