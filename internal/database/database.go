package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"developer-portal-backend/internal/database/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Options struct {
	LogLevel        logger.LogLevel
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	AutoMigrate     bool
}

// Initialize opens a Postgres connection and creates the schema from GORM models.
// Simplified single-phase AutoMigrate since cyclic foreign keys were removed.
func Initialize(dsn string, opts *Options) (*gorm.DB, error) {
	log.Print("Initializing database...")
	// Defaults
	if opts == nil {
		opts = &Options{}
	}
	if opts.LogLevel == 0 {
		opts.LogLevel = logger.Error
	}
	if opts.MaxOpenConns == 0 {
		opts.MaxOpenConns = 20
	}
	if opts.MaxIdleConns == 0 {
		opts.MaxIdleConns = 10
	}
	if opts.ConnMaxLifetime == 0 {
		opts.ConnMaxLifetime = 30 * time.Minute
	}
	if opts.ConnMaxIdleTime == 0 {
		opts.ConnMaxIdleTime = 10 * time.Minute
	}
	if !opts.AutoMigrate {
		opts.AutoMigrate = true
	}

	dataDir, err := resolveDataDir()
	if err != nil {
		return nil, err
	}
	// Open DB
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(opts.LogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
		sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(opts.ConnMaxIdleTime)
	}

	// Ensure required extension for UUID generation (used by BaseModel default gen_random_uuid())
	_ = db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error

	// AutoMigrate all models (no cycles)
	if opts.AutoMigrate {
		all := []interface{}{
			&models.Organization{},
			&models.Group{},
			&models.User{},
			&models.Team{},
			&models.Documentation{},
			&models.Landscape{},
			&models.Project{},
			&models.Component{},
			&models.Category{},
			&models.Link{},
			&models.Plugin{},
			&models.Token{},
		}
		if err := db.AutoMigrate(all...); err != nil {
			return nil, fmt.Errorf("auto-migrate: %w", err)
		}

	}

	if err := CreateIndexes(db); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}
	if err := ErrorsFix(db); err != nil {
		return nil, fmt.Errorf("failed to fix errors in database: %w", err)
	}

	if err := InitDataFromYAMLs(db, dataDir); err != nil {
		return nil, fmt.Errorf("failed init data from YAML files: %w", err)
	}
	log.Print("Initializing database done.")
	return db, nil
}

func resolveDataDir() (string, error) {
	// Try common locations to accommodate local runs, tests, and containers
	candidates := []string{
		"scripts/data",
		"/app/scripts/data",
	}
	// Walk up from current working directory to find scripts/data
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 5 && dir != "/" && dir != ""; i++ {
			p := filepath.Join(dir, "scripts", "data")
			candidates = append(candidates, p)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("data directory not found: tried %v", candidates)
}

func CreateIndexes(db *gorm.DB) error {
	// Ensure unique index on organizations.name
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS organizations_name_unique ON organizations (name)`).Error; err != nil {
		return fmt.Errorf("create unique index organizations.name: %w", err)
	}
	// Ensure unique index on group.name with org_id
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS groups_name_org_id_unique ON groups (name, org_id)`).Error; err != nil {
		return fmt.Errorf("create unique index groups.name+organization_id: %w", err)
	}
	// Ensure unique index on team.name with group_id
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS teams_name_group_id_unique ON teams (name, group_id)`).Error; err != nil {
		return fmt.Errorf("create unique index teams.name+group_id: %w", err)
	}
	// Ensure unique index on project.name
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS projects_name_unique ON projects (name)`).Error; err != nil {
		return fmt.Errorf("create unique index projects.name: %w", err)
	}
	// Ensure unique index on landscape.name and project_id
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS landscapes_name_project_id_unique ON landscapes (name, project_id)`).Error; err != nil {
		return fmt.Errorf("create unique index landscapes.name+project_id: %w", err)
	}
	// Ensure unique index on component.name and project_id
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS components_name_project_id_unique ON components (name, project_id)`).Error; err != nil {
		return fmt.Errorf("create unique index components.name+project_id: %w", err)
	}
	// Ensure unique index on link.name and category_id
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS links_name_category_id_unique ON links (name, category_id)`).Error; err != nil {
		return fmt.Errorf("create unique index links.name+category_id: %w", err)
	}
	// Ensure unique index on category.name
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS categories_name_unique ON categories (name)`).Error; err != nil {
		return fmt.Errorf("create unique index categories.name: %w", err)
	}

	return nil
}

// ErrorsFix removes known erroneous data from early versions of the database.
// most of those fixes can be removed after a version was deployed to 'dev' and 'prod' environments, as they are one-time fixes.
func ErrorsFix(db *gorm.DB) error {
	// remove components which belong to project 'internal' - this was a test project created in early versions:
	if err := db.Exec(`DELETE FROM components WHERE project_id IN (SELECT id FROM projects WHERE name = ?)`, "internal").Error; err != nil {
		return err
	}
	// remove projects with name 'noe' or 'internal' - these were test projects created in early versions:
	if err := db.Exec(`DELETE FROM projects WHERE name IN (?,?)`, "noe", "internal").Error; err != nil {
		return err
	}
	// remove 'health-success-regex' metadata entries from all projects - these had invalid data structures in early versions:
	if err := db.Exec(`UPDATE projects SET metadata = jsonb_strip_nulls(metadata - 'health-success-regex') WHERE metadata ? 'health-success-regex'`).Error; err != nil {
		return err
	}
	return nil
}
