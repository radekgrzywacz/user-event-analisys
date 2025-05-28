package user

type Store interface {
	Save(User) error
	GetAvailableUsers(neededCount int) ([]User, error)
}

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Country   string `json:"country"`
	IP        string `json:"ip"`
	UserAgent string `json:"user-agent"`
}
