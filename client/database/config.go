package database

const (
	TypeMySQL    = "mysql"
	TypePostgres = "postgres"
)

type Config struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // "mysql" (default) or "postgres"
	Database string   `json:"database"`
	User     string   `json:"user"`
	Password string   `json:"password"`
	Masters  []string `json:"masters"`
	Slaves   []string `json:"slaves"`
}
