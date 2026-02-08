package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/config"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

const (
	// Connection pool settings
	maxIdleConns    = 10
	maxOpenConns    = 100
	connMaxLifetime = time.Hour
	connMaxIdleTime = 10 * time.Minute
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=10",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	if err := db.AutoMigrate(
		&models.GovernmentFunding{},
		&models.GovernmentFundingPeriod{},
		&models.GovernmentFundingProperty{},
		&models.Organization{},
		&models.User{},
		&models.Group{},
		&models.Section{},
		&models.UserGroup{},
		&models.Employee{},
		&models.EmployeeContract{},
		&models.Child{},
		&models.ChildContract{},
		&models.AuditLog{},
		&models.PayPlan{},
		&models.PayPlanPeriod{},
		&models.PayPlanEntry{},
		&models.Attendance{},
	); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}
