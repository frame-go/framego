package cache

type Config struct {
	// Name of the client.
	Name string

	// Type of Cache client. Choices: "redis"
	Type string

	// host:port address.
	Address string

	// Optional. Use the specified Username to authenticate the current connection.
	Username string

	// Optional. Password for authentication.
	Password string

	// Database to be selected after connecting to the server.
	DB uint32
}
