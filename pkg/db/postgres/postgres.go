package postgres

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/NKTKLN/todo-api/pkg/db"
)

type PDB struct {
	DB *gorm.DB
}

// Connecting to a PostgreSQL database
func Connect(postgresHost, postgresUser, postgresPassword, postgresDBName string, postgresPort int) (db.PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		postgresHost, postgresPort, postgresUser, postgresPassword, postgresDBName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		return nil, err
	}

	return &PDB{DB: db}, nil
}
