package entity

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Country   string `json:"country"`
	IP        string `json:"ip"`
	UserAgent string `json:"user-agent"`
}
