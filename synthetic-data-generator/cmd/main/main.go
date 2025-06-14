package main

import (
	"log"
	"synthetic-data-generator/internal/config"
	"synthetic-data-generator/internal/event"
	"synthetic-data-generator/internal/initializer"
	"synthetic-data-generator/internal/user"
	"time"

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

	userRepo := user.NewUserRepository(config.DB)
	userService := user.NewUserService(faker, userRepo)
	generator := event.NewGenerator(userService, faker)

	for {
		for _, user := range users {
			err := generator.RunBotActivityScenario(user.ID, 1)
			if err != nil {
				log.Printf("Error in RunBotActivityScenario: %v", err)
			}
			log.Printf("Event sent for user ID %d", user.ID)
			time.Sleep(3 * time.Second)
		}
	}
}
