package database

type Config struct {
	Name     string   `json:"name"`
	Database string   `json:"database"`
	User     string   `json:"user"`
	Password string   `json:"password"`
	Masters  []string `json:"masters"`
	Slaves   []string `json:"slaves"`
}
