//loads config.yaml + env overrides. The DBDriver flag selects the driver at runtime,

package config

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper" // Viper library to read config file + env variables
)

// ==============================
// Config is a struct that mirrors the shape of our expected configuration.
// Viper will unmarshal values from YAML/env into these fields.

type Config struct {
	AppName    string `mapstructure:"app_name"`
	Env        string `mapstructure:"env"`         // dev|staging|prod
	HTTPPort   string `mapstructure:"http_port"`   // "8080"
	JWTSecret  string `mapstructure:"jwt_secret"`  // strong secret
	JWTExpires string `mapstructure:"jwt_expires"` // Token lifetime parsed by time.ParseDuration, e.g., "72h".

	//JWTExpires time.Duration `mapstructure:"jwt_expires"`   // "72h" X X X X X X X X X X X 

	// Database settings.select a driver then read its DSN/Path accordingly.
	//
	DBDriver     string `mapstructure:"db_driver"`     // mysql|postgres|sqlite|sqlserver
	MySQLDSN     string `mapstructure:"mysql_dsn"`     // user:pass@tcp(host:3306)/db?parseTime=true
	PostgresDSN  string `mapstructure:"postgres_dsn"`  // host=... user=... password=... dbname=... sslmode=disable
	SQLitePath   string `mapstructure:"sqlite_path"`   // "app.db"
	SQLServerDSN string `mapstructure:"sqlserver_dsn"` // sqlserver://user:pass@host:1433?database=DB

	//
	//

	RedisAddr string `mapstructure:"redis_addr"`     // "localhost:6379" // Host:port for Redis server.
	RedisDB   int    `mapstructure:"redis_db"`       // Redis logical DB number
	RedisPass string `mapstructure:"redis_password"` // Redis password (if any)
}

// expose parsed duration globally
var JWTExpiryDuration time.Duration

func Load() *Config {
	v := viper.New()                                   // Create a new Viper instance (isolated, not global).
	v.SetConfigName("config")                          // Expect a file named "config.(yaml|yml|json...)".
	v.SetConfigType("yaml")                            // Tell Viper we will load YAML if file is present.
	v.AddConfigPath(".")                               // Look for config.yaml in the project root.
	v.AddConfigPath("./config")                        // Also allow a config inside ./config (optional).
	v.SetEnvPrefix("APP")                              // Prefix for env overrides like APP_HTTP_PORT.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // Convert nested keys to ENV_STYLE
	v.AutomaticEnv()                                   // Enable reading from environment variables automatically.

	// defaults (safe for local)
	v.SetDefault("app_name", "HelmyTask")        // Default app name.
	v.SetDefault("env", "dev")                   // Default environment.
	v.SetDefault("http_port", "8080")            //default http portt
	v.SetDefault("jwt_expires", "72h")           // default jwt lifetime
	v.SetDefault("db_driver", "mysql")           //default to MySql(can be also : postgres | sqlite || sqlserver)
	v.SetDefault("sqlite_path", "app.db")        //// Default sqlite file path if sqlite is used.
	v.SetDefault("redis_addr", "localhost:6379") // Default Redis address.
	v.SetDefault("redis_db", 0)                  // Use Redis DB 0 by default.

	// Try to read config file; if not found, proceed with defaults + env vars.

	if err := v.ReadInConfig(); err != nil {
		log.Printf("[config] no config file found, using defaults/env: %v", err)
	}
//
	// Create an empty Config struct to fill.
	var c Config
	// Unmarshal Viper’s aggregated settings (defaults + file + env) into the struct.

	if err := v.Unmarshal(&c); err != nil {
		log.Fatalf("[config] unmarshal error: %v", err) // Fatal if we can’t parse.
	}

	// parse jwt_expires string into time.Duration
	d, err := time.ParseDuration(c.JWTExpires)

	if err != nil {
		log.Fatalf("[config] invalid jwt_expires value: %v", err)
	}
	JWTExpiryDuration = d

	return &c // Return a pointer so caller shares the same object.

}
