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

func (g *Generator) RunBrowseAndPurchaseScenario(userID int) error {
	sessionId := uuid.NewString()

	for i := 0; i < g.faker.Number(2, 6); i++ {
		ev, err := g.CreateGoodEvent(userID, EventOther)
		if err != nil {
			return err
		}
		ev.SessionId = sessionId
		ev.Additional["action"] = "product_view"
		ev.Additional["category"] = g.faker.RandomString([]string{"Electronics", "Home", "Books", "Toys"})
		ev.Additional["product_id"] = g.faker.UUID()
		if err := g.SendEvent(ev); err != nil {
			return err
		}
		time.Sleep(time.Duration(g.faker.Number(100, 500)) * time.Millisecond)
	}

	add, err := g.CreateGoodEvent(userID, EventOther)
	if err != nil {
		return err
	}
	add.SessionId = sessionId
	add.Additional["action"] = "add_to_cart"
	add.Additional["product_id"] = g.faker.UUID()
	if err := g.SendEvent(add); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	payment, err := g.CreateGoodEvent(userID, EventPayment)
	if err != nil {
		return err
	}
	payment.SessionId = sessionId
	payment.Additional["value"] = g.faker.Price(10, 400)
	payment.Additional["currency"] = "EUR"
	payment.Additional["category"] = "Retail"
	if err := g.SendEvent(payment); err != nil {
		return err
	}

	logout, err := g.CreateGoodEvent(userID, EventLogout)
	if err != nil {
		return err
	}
	logout.SessionId = sessionId
	if err := g.SendEvent(logout); err != nil {
		return err
	}

	return nil
}

func (g *Generator) RunPaymentRetryScenario(userID int) error {
	sessionId := uuid.NewString()

	login, _ := g.CreateGoodEvent(userID, EventLogin)
	login.SessionId = sessionId
	g.SendEvent(login)
	time.Sleep(300 * time.Millisecond)

	p1, _ := g.CreateRandomEvent(userID, EventPayment)
	p1.SessionId = sessionId
	p1.Additional["value"] = g.faker.Price(20, 500)
	p1.Additional["payment_status"] = "failed"
	if err := g.SendEvent(p1); err != nil {
		return err
	}

	time.Sleep(700 * time.Millisecond)

	p2, _ := g.CreateGoodEvent(userID, EventPayment)
	p2.SessionId = sessionId
	p2.Additional["value"] = p1.Additional["value"]
	p2.Additional["payment_status"] = "success"
	if err := g.SendEvent(p2); err != nil {
		return err
	}

	logout, _ := g.CreateGoodEvent(userID, EventLogout)
	logout.SessionId = sessionId
	g.SendEvent(logout)

	return nil
}

func (g *Generator) RunSubscriptionRenewalScenario(userID int) error {
	sessionId := uuid.NewString()

	login, _ := g.CreateGoodEvent(userID, EventLogin)
	login.SessionId = sessionId
	g.SendEvent(login)
	time.Sleep(200 * time.Millisecond)

	payment, _ := g.CreateGoodEvent(userID, EventPayment)
	payment.SessionId = sessionId
	payment.Additional["value"] = 9.99
	payment.Additional["currency"] = "EUR"
	payment.Additional["category"] = "Subscription"
	payment.Additional["recurring"] = true
	if err := g.SendEvent(payment); err != nil {
		return err
	}

	receipt, _ := g.CreateGoodEvent(userID, EventOther)
	receipt.SessionId = sessionId
	receipt.Additional["action"] = "subscription_receipt"
	g.SendEvent(receipt)

	logout, _ := g.CreateGoodEvent(userID, EventLogout)
	logout.SessionId = sessionId
	g.SendEvent(logout)
	return nil
}

func (g *Generator) RunProfileUpdateScenario(userID int) error {
	sessionId := uuid.NewString()
	login, _ := g.CreateGoodEvent(userID, EventLogin)
	login.SessionId = sessionId
	g.SendEvent(login)
	time.Sleep(300 * time.Millisecond)

	up, _ := g.CreateGoodEvent(userID, EventOther)
	up.SessionId = sessionId
	up.Additional["action"] = "profile_update"
	up.Additional["field_changed"] = g.faker.RandomString([]string{"email", "phone", "address", "password"})
	if err := g.SendEvent(up); err != nil {
		return err
	}

	logout, _ := g.CreateGoodEvent(userID, EventLogout)
	logout.SessionId = sessionId
	g.SendEvent(logout)
	return nil
}

func (g *Generator) RunLongShoppingSession(userID int) error {
	sessionId := uuid.NewString()
	login, _ := g.CreateGoodEvent(userID, EventLogin)
	login.SessionId = sessionId
	g.SendEvent(login)

	for i := 0; i < 10; i++ {
		v, _ := g.CreateGoodEvent(userID, EventOther)
		v.SessionId = sessionId
		v.Additional["action"] = "browse"
		v.Additional["category"] = g.faker.RandomString([]string{"Books", "Electronics", "Clothes"})
		g.SendEvent(v)
		time.Sleep(200 * time.Millisecond)

		if g.faker.Number(0, 10) > 7 {
			pay, _ := g.CreateGoodEvent(userID, EventPayment)
			pay.SessionId = sessionId
			pay.Additional["value"] = g.faker.Price(5, 200)
			pay.Additional["category"] = "Retail"
			g.SendEvent(pay)
		}
	}

	logout, _ := g.CreateGoodEvent(userID, EventLogout)
	logout.SessionId = sessionId
	g.SendEvent(logout)
	return nil
}
