package event

import (
	"fmt"
	"time"
)

func (g *Generator) RunNormalUserScenario(userID int) error {
	login, err := g.CreateGoodEvent(userID, EventLogin)
	if err != nil {
		return err
	}
	if err := g.SendEvent(login); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	payment, err := g.CreateGoodEvent(userID, EventPayment)
	if err != nil {
		return err
	}
	if err := g.SendEvent(payment); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	logout, err := g.CreateGoodEvent(userID, EventLogout)
	if err != nil {
		return err
	}
	return g.SendEvent(logout)
}


func (g *Generator) RunAnomalousLoginScenario(userID int) error {
	loginAnomaly, err := g.CreateRandomEvent(userID, EventLogin)
	if err != nil {
	  return err
	}
	g.SendEvent(loginAnomaly)
	
	return nil
}

func (g *Generator) RunBruteForceScenario(userID int, attempts int) error {
	for i := 0; i < attempts; i++ {
		event, err := g.CreateRandomEvent(userID, EventFailedLogin)
		if err != nil {
		  return err
		}
		g.SendEvent(event)
		time.Sleep(200 * time.Millisecond)
	}
	
	return nil
}

func (g *Generator) RunAccountTakeoverScenario(userID int) error {
	login, err := g.CreateGoodEvent(userID, EventLogin)
	if err != nil {
	  return err
	}
	g.SendEvent(login)

	time.Sleep(1 * time.Second)

	reset, err := g.CreateRandomEvent(userID, EventPasswordReset)
	if err != nil {
	  return err
	}
	g.SendEvent(reset)

	time.Sleep(1 * time.Second)

	loginAfterReset, err := g.CreateRandomEvent(userID, EventLogin)
	if err != nil {
	  return err
	}
	g.SendEvent(loginAfterReset)
	
	return nil
}

func (g *Generator) RunBotActivityScenario(userID int, count int) error {
	for range count {
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

func (g *Generator) RunFraudTransactionScenario(userID int) error {
	login, err := g.CreateGoodEvent(userID, EventLogin)
	if err != nil {
		return err
	}
	g.SendEvent(login)

	time.Sleep(1 * time.Second)

	payment, err := g.CreateRandomEvent(userID, EventPayment)
	if err != nil {
		return err
	}
	return g.SendEvent(payment)
}

