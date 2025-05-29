package main

import (
	"fmt"
	"log"
	"synthetic-data-generator/internal/config"
	"synthetic-data-generator/internal/initializer"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../../.env")	
	config := config.SetupConfig()
	faker := gofakeit.New(123)
	neededUsers := config.Flags.UsersCount
	

	users, err := initializer.CreateUsersIfNeeded(neededUsers, config.DB, faker)
	if err != nil {
		log.Fatalf("Error creating new users: %v", err)
	}

	for index, user := range users {
		fmt.Printf("%s: %s \n", index, user)
	}
}
