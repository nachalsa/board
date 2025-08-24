package config

import (
	"os"
	"strconv"
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
			UploadsDir:        getEnv("UPLOADS_DIR", "files/uploads"),
			UploadsDeletedDir: getEnv("UPLOADS_DELETED_DIR", "files/deleted"),
			MaxFileSize:       getEnvInt64("MAX_FILE_SIZE_MB", 500) * 1024 * 1024,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
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

func (c *Config) GetMaxFileSizeMB() int64 {
	return c.File.MaxFileSize / (1024 * 1024)
}

func (c *Config) GetMaxFileSizeText() string {
	sizeMB := c.GetMaxFileSizeMB()
	return strconv.FormatInt(sizeMB, 10) + "MB"
}
