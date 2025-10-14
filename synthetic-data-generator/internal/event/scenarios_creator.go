package event

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (g *Generator) RunNormalUserScenario(userID int) error {
	sessionId := uuid.NewString()
	login, err := g.CreateGoodEvent(userID, EventLogin)
	if err != nil {
		return err
	}
	login.SessionId = sessionId
	if err := g.SendEvent(login); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	payment, err := g.CreateGoodEvent(userID, EventPayment)
	if err != nil {
		return err
	}
	payment.SessionId = sessionId
	if err := g.SendEvent(payment); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)

	logout, err := g.CreateGoodEvent(userID, EventLogout)
	logout.SessionId = sessionId
	if err != nil {
		return err
	}
	if err := g.SendEvent(logout); err != nil {
		return err
	}

	return nil
}

func (g *Generator) RunAnomalousLoginScenario(userID int) error {
	loginAnomaly, err := g.CreateRandomEvent(userID, EventLogin)
	if err != nil {
		return err
	}
	if err := g.SendEvent(loginAnomaly); err != nil {
		return fmt.Errorf("sending anomalous login event: %w", err)
	}

	return nil
}

func (g *Generator) RunBruteForceScenario(userID int, attempts int) error {
	for i := 0; i < attempts; i++ {
		ev, err := g.CreateRandomEvent(userID, EventFailedLogin)
		if err != nil {
			return err
		}
		if err := g.SendEvent(ev); err != nil {
			return fmt.Errorf("sending failed-login event #%d: %w", i+1, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func (g *Generator) RunAccountTakeoverScenario(userID int) error {
	sessionId := uuid.NewString()
	login, err := g.CreateGoodEvent(userID, EventLogin)
	if err != nil {
		return err
	}
	login.SessionId = sessionId
	if err := g.SendEvent(login); err != nil {
		return fmt.Errorf("sending initial login: %w", err)
	}

	time.Sleep(1 * time.Second)

	reset, err := g.CreateRandomEvent(userID, EventPasswordReset)
	if err != nil {
		return err
	}
	reset.SessionId = sessionId
	if err := g.SendEvent(reset); err != nil {
		return fmt.Errorf("sending password reset: %w", err)
	}

	time.Sleep(1 * time.Second)

	loginAfterReset, err := g.CreateRandomEvent(userID, EventLogin)
	if err != nil {
		return err
	}
	loginAfterReset.SessionId = sessionId
	if err := g.SendEvent(loginAfterReset); err != nil {
		return fmt.Errorf("sending login after reset: %w", err)
	}

	return nil
}

func (g *Generator) RunBotActivityScenario(userID int, count int) error {
	sessionId := uuid.NewString()
	types := []EventType{EventLogin, EventLogout, EventPayment, EventOther}

	for i := 0; i < count; i++ {
		idx := g.faker.Number(0, len(types)-1)
		evType := types[idx]

		ev, err := g.CreateRandomEvent(userID, evType)
		if err != nil {
			return fmt.Errorf("creating bot event #%d: %w", i+1, err)
		}
		ev.SessionId = sessionId
		if err := g.SendEvent(ev); err != nil {
			return fmt.Errorf("sending bot event #%d: %w", i+1, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	return nil
}

func (g *Generator) RunFraudTransactionScenario(userID int) error {
	sessionId := uuid.NewString()
	login, err := g.CreateGoodEvent(userID, EventLogin)
	if err != nil {
		return err
	}
	login.SessionId = sessionId
	if err := g.SendEvent(login); err != nil {
		return fmt.Errorf("sending login: %w", err)
	}

	time.Sleep(1 * time.Second)

	payment, err := g.CreateRandomEvent(userID, EventPayment)
	if err != nil {
		return err
	}
	payment.SessionId = sessionId
	if err := g.SendEvent(payment); err != nil {
		return fmt.Errorf("sending fraudulent payment: %w", err)
	}

	return nil
}
