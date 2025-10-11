package main

import (
	"log"
	"math/rand/v2"
	"sync"
	"synthetic-data-generator/internal/config"
	"synthetic-data-generator/internal/event"
	"synthetic-data-generator/internal/initializer"
	"synthetic-data-generator/internal/user"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/joho/godotenv"
)

func runRandomAnomaly(g *event.Generator, userID int) {
	scenarios := []func(int) error{
		g.RunAnomalousLoginScenario,
		func(id int) error { return g.RunBruteForceScenario(id, 5) },
		g.RunAccountTakeoverScenario,
		func(id int) error { return g.RunBotActivityScenario(id, 3) },
		g.RunFraudTransactionScenario,
	}

	selected := scenarios[rand.IntN(len(scenarios))]
	if err := selected(userID); err != nil {
		log.Printf("❌ Anomaly scenario error for user %d: %v", userID, err)
	}
}

func simulateUsers(users []user.User, config *config.Config, generator *event.Generator, endTime time.Time, infinite bool) {
	sem := make(chan struct{}, config.Flags.Concurrency)
	var wg sync.WaitGroup

	for infinite || time.Now().Before(endTime) {
		for i := 0; i < len(users); i++ {
			currentUser := users[rand.IntN(len(users))]
			sem <- struct{}{}
			wg.Add(1)

			go func(u user.User) {
				defer func() {
					<-sem
					wg.Done()
				}()

				if rand.Float64() < config.Flags.AnomalyRate {
					runRandomAnomaly(generator, u.ID)
					log.Printf("Anomaly sent for user ID %d", u.ID)
				} else {
					if err := generator.RunNormalUserScenario(u.ID); err != nil {
						log.Printf("Error running normal scenario for user %d: %v", u.ID, err)
					}
					log.Printf("Event sent for user ID %d", u.ID)
				}
			}(currentUser)

			time.Sleep(300 * time.Millisecond)
		}
	}
	wg.Wait()
}

func setupGeneratorAndUsers(cfg *config.Config) ([]user.User, *event.Generator, error) {
	faker := gofakeit.New(uint64(time.Now().Unix()))

	users, err := initializer.CreateUsersIfNeeded(cfg.Flags.UsersCount, cfg.DB, faker)
	if err != nil {
		return nil, nil, err
	}

	userRepo := user.NewUserRepository(cfg.DB)
	userService := user.NewUserService(faker, userRepo)
	generator := event.NewGenerator(userService, faker)

	return users, generator, nil
}

func main() {
	_ = godotenv.Load("../../.env")
	config := config.SetupConfig()
	log.Print(config.Flags)

	users, generator, err := setupGeneratorAndUsers(&config)
	if err != nil {
		log.Fatalf("❌ Error creating users and generator: %v", err)
	}

	duration := time.Duration(config.Flags.DurationInSeconds) * time.Second
	infinite := duration <= 0
	var endTime time.Time
	if !infinite {
		endTime = time.Now().Add(duration)
	}

	log.Printf("🚀 Starting simulation for %v seconds with %d users", duration.Seconds(), len(users))

	simulateUsers(users, &config, generator, endTime, infinite)

	log.Println("✅ Simulation complete")
}
