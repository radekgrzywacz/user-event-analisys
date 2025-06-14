package event

import (
	"fmt"
	"time"
)

func (g *Generator) RunNormalUserScenario(userID int) {
	login, _ := g.CreateGoodEvent(userID, EventLogin)
	g.SendEvent(login)

	time.Sleep(1 * time.Second)

	payment, _ := g.CreateGoodEvent(userID, EventPayment)
	g.SendEvent(payment)

	time.Sleep(1 * time.Second)

	logout, _ := g.CreateGoodEvent(userID, EventLogout)
	g.SendEvent(logout)
}

func (g *Generator) RunAnomalousLoginScenario(userID int) {
	loginAnomaly, _ := g.CreateRandomEvent(userID, EventLogin)
	g.SendEvent(loginAnomaly)
}

func (g *Generator) RunBruteForceScenario(userID int, attempts int) {
	for i := 0; i < attempts; i++ {
		event, _ := g.CreateRandomEvent(userID, EventFailedLogin)
		g.SendEvent(event)
		time.Sleep(200 * time.Millisecond)
	}
}

func (g *Generator) RunAccountTakeoverScenario(userID int) {
	login, _ := g.CreateGoodEvent(userID, EventLogin)
	g.SendEvent(login)

	time.Sleep(1 * time.Second)

	reset, _ := g.CreateRandomEvent(userID, EventPasswordReset)
	g.SendEvent(reset)

	time.Sleep(1 * time.Second)

	loginAfterReset, _ := g.CreateRandomEvent(userID, EventLogin)
	g.SendEvent(loginAfterReset)
}

func (g *Generator) RunBotActivityScenario(userID int, count int) error {
	for i := 0; i < count; i++ {
		evType := EventType([]string{
			string(EventLogin),
			string(EventLogout),
			string(EventPayment),
			string(EventOther),
		}[g.faker.Number(0, 3)])

		event, _ := g.CreateRandomEvent(userID, EventType(evType))
		if err := g.SendEvent(event); err != nil {
			return fmt.Errorf("Error sending bot event: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	return nil
}
