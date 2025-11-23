package testutils

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"developer-portal-backend/internal/config"
	"developer-portal-backend/internal/database"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for readiness ping
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// ------------------------------
// Shared, process-wide resources
// ------------------------------
var (
	sharedOnce     sync.Once
	sharedInitErr  error
	sharedPool     *dockertest.Pool
	sharedResource *dockertest.Resource
	sharedDB       *gorm.DB
	sharedConfig   *config.Config
)

// ------------------------------
// Base suite types
// ------------------------------
type BaseTestSuite struct {
	suite.Suite
	DB       *gorm.DB
	Config   *config.Config
	pool     *dockertest.Pool
	resource *dockertest.Resource
}

type HandlerTestSuite struct {
	*BaseTestSuite
	Router interface{} // replace with *gin.Engine when you wire routes
}

type ServiceTestSuite struct {
	*BaseTestSuite
	Mocks map[string]interface{}
}

type RepositoryTestSuite struct {
	*BaseTestSuite
	Repositories map[string]interface{}
}

// ------------------------------
// Public helpers
// ------------------------------

// SetupTestSuite initializes (once) the shared Postgres container and returns a per-suite wrapper.
// Call this in your tests before using the DB.
func SetupTestSuite(t *testing.T) *BaseTestSuite {
	sharedOnce.Do(func() { sharedInitErr = initSharedPGContainer() })
	if sharedInitErr != nil {
		t.Fatalf("failed to initialize shared test container: %v", sharedInitErr)
	}
	return &BaseTestSuite{
		DB:       sharedDB,
		Config:   sharedConfig,
		pool:     sharedPool,
		resource: sharedResource,
	}
}

// CleanupSharedContainer tears down Docker resources when the whole test run ends.
// This is automatically called by TestMain in main_test.go
func CleanupSharedContainer() {
	log.Println("Starting Docker container cleanup...")
	if sharedDB != nil {
		if sqlDB, err := sharedDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	if sharedPool != nil && sharedResource != nil {
		log.Printf("Purging Docker container: %s", sharedResource.Container.Name)
		if err := sharedPool.Purge(sharedResource); err != nil {
			log.Printf("WARN: could not purge shared resource: %v", err)
		} else {
			log.Println("Successfully purged Docker container")
		}
		// Reset shared variables
		sharedResource = nil
		sharedPool = nil
		sharedDB = nil
	}
}

// RunWithTestSuite is a convenience wrapper to run a function with a ready suite.
func RunWithTestSuite(t *testing.T, testFunc func(*BaseTestSuite)) {
	s := SetupTestSuite(t)
	defer s.TeardownTestSuite()
	testFunc(s)
}

// ------------------------------
// Suite lifecycle hooks
// ------------------------------

func (s *BaseTestSuite) SetupTest()    { s.CleanTestDB() }
func (s *BaseTestSuite) TearDownTest() { s.CleanTestDB() }

// TeardownTestSuite is per *suite* (not process). We only clean DB here;
// Docker container persists across suites for speed.
func (s *BaseTestSuite) TeardownTestSuite() { s.CleanTestDB() }

// CleanTestDB truncates known tables if they exist. Safe even if schema changes.
func (s *BaseTestSuite) CleanTestDB() {
	if s.DB == nil {
		return
	}
	tables := []string{
		"links",
		"components",
		"landscapes",
		"projects",
		"categories",
		"users",
		"teams",
		"groups",
		"organizations",
	}
	m := s.DB.Migrator()
	s.DB.Exec(`SET session_replication_role = replica;`)
	for _, t := range tables {
		if m.HasTable(t) {
			s.DB.Exec(`TRUNCATE TABLE "` + t + `" RESTART IDENTITY CASCADE;`)
		}
	}
	s.DB.Exec(`SET session_replication_role = DEFAULT;`)
}

// ------------------------------
// Shared Postgres container init
// ------------------------------

func initSharedPGContainer() error {
	// 1) Create Docker pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("could not connect to docker: %w", err)
	}
	sharedPool = pool

	// 2) Run Postgres
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15-alpine",
		Env: []string{
			"POSTGRES_PASSWORD=testpass",
			"POSTGRES_USER=testuser",
			"POSTGRES_DB=testdb",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return fmt.Errorf("could not start postgres: %w", err)
	}
	sharedResource = resource

	// 3) Build DSN
	hostPort := resource.GetPort("5432/tcp")
	dsn := fmt.Sprintf("postgres://testuser:testpass@127.0.0.1:%s/testdb?sslmode=disable", hostPort)

	// 4) Wait for Postgres to be ready, then init GORM (which runs migrations)
	pool.MaxWait = 2 * time.Minute
	if err := pool.Retry(func() error {
		// 4a) Ping with database/sql first (fast readiness)
		std, err := sql.Open("pgx",
			fmt.Sprintf("host=127.0.0.1 port=%s user=testuser password=testpass dbname=testdb sslmode=disable", hostPort),
		)
		if err != nil {
			return err
		}
		defer std.Close()

		deadline := time.Now().Add(15 * time.Second)
		for {
			if err := std.Ping(); err == nil {
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("postgres not ready to accept connections")
			}
			time.Sleep(250 * time.Millisecond)
		}

		// 4b) Now initialize GORM (your database.Initialize does two-phase migration)
		gdb, err := database.Initialize(dsn, nil)
		if err != nil {
			return err
		}
		// final sanity ping
		if sqlDB, err := gdb.DB(); err != nil {
			return err
		} else if err := sqlDB.Ping(); err != nil {
			return err
		}
		sharedDB = gdb
		return nil
	}); err != nil {
		return fmt.Errorf("could not connect to docker database: %w", err)
	}

	// 5) Build a shared config (if your app/tests need config)
	sharedConfig = &config.Config{
		DatabaseURL: dsn,
		Port:        "8080",
		LogLevel:    "debug",
		Environment: "test",
	}

	log.Printf("✅ Shared Postgres ready on %s", hostPort)
	logExistingTables(sharedDB) // optional: helps debug schema presence early
	return nil
}

// Optional diagnostics: list public tables after init
func logExistingTables(db *gorm.DB) {
	type row struct{ Tablename string }
	var rows []row
	if err := db.Raw(
		`SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`,
	).Scan(&rows).Error; err == nil {
		names := make([]string, 0, len(rows))
		for _, r := range rows {
			names = append(names, r.Tablename)
		}
		log.Printf("Public tables: %v", names)
	}
}
