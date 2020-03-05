package db

const InMemory = "sqlite3://:memory:"

func NewInMemoryDBForTests(logMode bool) {
	NewDb(DatabaseOptions{
		Connection: InMemory,
		LogMode:    logMode,
	})
}
