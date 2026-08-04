package config

type Config struct {
	Host  string
	Port  int
	DB    DatabaseConfig
	Cache CacheConfig
}

// Specify only one of these options:
type DatabaseConfig struct {
	Sqlite3 *Sqlite3Config
	// Postgresql *PosgresqlConfig
}

type Sqlite3Config struct {
	// Path to the .db file
	// or ":memory:" for in-memory DB.
	// Leave empty for temporary storing
	Storage string `yaml:"storage"`
	// "shared" for shared use
	// or "private" for private use
	CacheAccess string `yaml:"cache_access"`
	// Pass true to enable Write Ahead of Logging
	WAL bool `yaml:"wal"`
}

// Only for redis for now
type CacheConfig struct {
	Host string
	Port int
}
