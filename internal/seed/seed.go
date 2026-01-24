package seed

import (
	"errors"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/config"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/rbac"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// SeedAdmin creates an initial admin user if SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD are set.
// If the user already exists, it will be skipped.
// The user will be assigned the superadmin role.
func SeedAdmin(cfg *config.Config, userStore *store.UserStore, enforcer *rbac.Enforcer) error {
	if cfg.SeedAdminEmail == "" || cfg.SeedAdminPassword == "" {
		slog.Info("Admin seeding skipped: SEED_ADMIN_EMAIL or SEED_ADMIN_PASSWORD not set")
		return nil
	}

	// Check if user already exists
	existingUser, err := userStore.FindByEmail(cfg.SeedAdminEmail)
	if err == nil && existingUser != nil {
		slog.Info("Admin user already exists", "email", cfg.SeedAdminEmail)

		// Ensure superadmin role is assigned
		if err := enforcer.AssignSuperAdmin(existingUser.ID); err != nil {
			slog.Warn("Failed to ensure superadmin role", "error", err)
		}
		return nil
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create admin user
	user := &models.User{
		Name:      cfg.SeedAdminName,
		Email:     cfg.SeedAdminEmail,
		Password:  string(hashedPassword),
		Active:    true,
		CreatedBy: "system",
	}

	if err := userStore.Create(user); err != nil {
		return err
	}

	slog.Info("Admin user created", "email", cfg.SeedAdminEmail, "id", user.ID)

	// Assign superadmin role
	if err := enforcer.AssignSuperAdmin(user.ID); err != nil {
		return err
	}

	slog.Info("Superadmin role assigned", "userId", user.ID)

	return nil
}
