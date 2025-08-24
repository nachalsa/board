package config

import (
	"os"
)

type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	File     FileConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type ServerConfig struct {
	Port      string
	AdminPort string
}

type FileConfig struct {
	UploadsDir        string
	UploadsDeletedDir string
	MaxFileSize       int64
}

func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", ""),
		},
		Server: ServerConfig{
			Port:      getEnv("SERVER_PORT", "8080"),
			AdminPort: getEnv("ADMIN_PORT", "8081"),
		},
		File: FileConfig{
			UploadsDir:        "files/uploads",
			UploadsDeletedDir: "files/deleted",
			MaxFileSize:       500 * 1024 * 1024, // 500MB
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) GetDatabaseURL() string {
	return "host=" + c.Database.Host +
		" port=" + c.Database.Port +
		" user=" + c.Database.User +
		" password=" + c.Database.Password +
		" dbname=" + c.Database.Name +
		" sslmode=disable"
}
