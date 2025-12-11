package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Environment       string `mapstructure:"ENVIRONMENT"`
	DeployEnvironment string `mapstructure:"DEPLOY_ENVIRONMENT"`
	Port              string `mapstructure:"PORT"`
	LogLevel          string `mapstructure:"LOG_LEVEL"`

	// Database configuration
	DatabaseURL      string `mapstructure:"DATABASE_URL"`
	DatabaseHost     string `mapstructure:"DB_HOST"`
	DatabasePort     string `mapstructure:"DB_PORT"`
	DatabaseUser     string `mapstructure:"DB_USER"`
	DatabasePassword string `mapstructure:"DB_PASSWORD"`
	DatabaseName     string `mapstructure:"DB_NAME"`
	DatabaseSSLMode  string `mapstructure:"DB_SSL_MODE"`

	// JWT configuration
	JWTSecret string `mapstructure:"JWT_SECRET"`

	// CORS configuration
	AllowedOrigins []string `mapstructure:"ALLOWED_ORIGINS"`

	// LDAP configuration
	LDAPHost               string `mapstructure:"LDAP_HOST"`
	LDAPPort               string `mapstructure:"LDAP_PORT"`
	LDAPBindDN             string `mapstructure:"LDAP_BIND_DN"`
	LDAPBindPW             string `mapstructure:"LDAP_BIND_PW"`
	LDAPBaseDN             string `mapstructure:"LDAP_BASE_DN"`
	LDAPInsecureSkipVerify bool   `mapstructure:"LDAP_INSECURE_SKIP_VERIFY"`
	LDAPTimeoutSec         int    `mapstructure:"LDAP_TIMEOUT_SEC"`

	// Jira configuration
	JiraDomain   string `mapstructure:"JIRA_DOMAIN"`
	JiraUser     string `mapstructure:"JIRA_USER"`
	JiraPassword string `mapstructure:"JIRA_PASSWORD"`

	// Sonar configuration
	SonarHost  string `mapstructure:"SONAR_HOST"`
	SonarToken string `mapstructure:"SONAR_TOKEN"`

	// Jenkins configuration
	JenkinsBaseURL            string `mapstructure:"JENKINS_BASE_URL"`
	JenkinsInsecureSkipVerify bool   `mapstructure:"JENKINS_INSECURE_SKIP_VERIFY"`

	// Monitoring service configuration
	MonitoringServiceURL string `mapstructure:"MONITORING_SERVICE_URL"`
}

// Load reads configuration from environment variables and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set default values
	setDefaults()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Override with environment variables
	viper.AutomaticEnv()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Build database URL if not provided
	if config.DatabaseURL == "" {
		config.DatabaseURL = buildDatabaseURL(&config)
	}

	// Validate required fields
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("DEPLOY_ENVIRONMENT", "local")
	viper.SetDefault("PORT", "7008")
	viper.SetDefault("LOG_LEVEL", "info")

	// Database defaults
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "developer_portal")
	viper.SetDefault("DB_SSL_MODE", "disable")

	// JWT defaults
	viper.SetDefault("JWT_SECRET", "your-secret-key-change-in-production")

	// CORS defaults
	viper.SetDefault("ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:8080"})

	// LDAP defaults
	viper.SetDefault("LDAP_HOST", "ldap.example.com")
	viper.SetDefault("LDAP_PORT", "636")
	viper.SetDefault("LDAP_BIND_DN", "CN=John Doe,OU=Users,DC=example,DC=com")
	viper.SetDefault("LDAP_BIND_PW", "SuperSecret123")
	viper.SetDefault("LDAP_BASE_DN", "DC=example,DC=com")
	viper.SetDefault("LDAP_INSECURE_SKIP_VERIFY", true)
	viper.SetDefault("LDAP_TIMEOUT_SEC", 10)

	// Jira defaults
	viper.SetDefault("JIRA_DOMAIN", "")
	viper.SetDefault("JIRA_USER", "")
	viper.SetDefault("JIRA_PASSWORD", "")

	// Sonar defaults
	viper.SetDefault("SONAR_HOST", "")
	viper.SetDefault("SONAR_TOKEN", "")

	// Jenkins defaults - production uses real JAAS URL pattern
	viper.SetDefault("JENKINS_BASE_URL", "https://{jaasName}.jaas-gcp.cloud.sap.corp")
	viper.SetDefault("JENKINS_INSECURE_SKIP_VERIFY", true)

	// Monitoring service defaults
	viper.SetDefault("MONITORING_SERVICE_URL", "http://localhost:8085")
}

func buildDatabaseURL(config *Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.DatabaseUser,
		config.DatabasePassword,
		config.DatabaseHost,
		config.DatabasePort,
		config.DatabaseName,
		config.DatabaseSSLMode,
	)
}

func validate(config *Config) error {
	if config.Environment == "production" {
		if config.JWTSecret == "your-secret-key-change-in-production" {
			return fmt.Errorf("JWT_SECRET must be set in production")
		}
	}

	if config.DatabaseName == "" {
		return fmt.Errorf("database name is required")
	}

	return nil
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
