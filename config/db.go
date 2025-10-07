//picks the GORM driver by DBDriver. No repository/service code changes needed when you change DB.

package config

import (
	"log"

	"HelmyTask/models" // Import our model(s) so we can auto-migrate schema.

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// GORM drivers (we open one depending on cfg.DBDriver).
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
)

// InitDB opens a database connection using the driver specified in config,
// configures GORM, and applies auto-migrations for our models.
func InitDB(cfg *Config) *gorm.DB {
	var (
		db  *gorm.DB //will hold the db connection
		err error    //error handler for opening connections
	)

	// Configure GORM’s logger to Warn to keep output readable (Info is very verbose).
 
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	switch cfg.DBDriver {
	case "mysql":
		if cfg.MySQLDSN == "" { // Ensure DSN is provided when driver is mysql.
			log.Fatal("[db] mysql selected but mysql_dsn empty")
		}
		db, err = gorm.Open(mysql.Open(cfg.MySQLDSN), gormCfg) //open connection
	case "postgres":
		if cfg.PostgresDSN == "" { //ensuree dsn is provided for postgres
			log.Fatal("[db] postgres selected but postgres_dsn empty")
		}
		db, err = gorm.Open(postgres.Open(cfg.PostgresDSN), gormCfg)
	case "sqlite":
		// SQLite only needs a file path; GORM will create file if missing.
		db, err = gorm.Open(sqlite.Open(cfg.SQLitePath), gormCfg)
	case "sqlserver":
		if cfg.SQLServerDSN == "" { // Ensure DSN is provided for SQL Server.
			log.Fatal("[db] sqlserver selected but sqlserver_dsn empty")
		}
		db, err = gorm.Open(sqlserver.Open(cfg.SQLServerDSN), gormCfg)
	default:
		log.Fatalf("[db] unknown DBDriver: %s", cfg.DBDriver) // Fail fast if driver is unsupported.

	}

	// If gorm.Open returned an error, abort.
	if err != nil {
		log.Fatalf("[db] connection error: %v", err)
	}

	
	// AutoMigrate creates or updates DB tables based on our struct definitions.
	// Safe for demos/starters; for real projects you may use migrations.
	// Migrate models (safe baseline)
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("[db] automigrate error: %v", err)
	}

	return db // Return the connected *gorm.DB to be injected into repositories.

}
