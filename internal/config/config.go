package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type (
	AppConfig struct {
		v        *viper.Viper
		Model    Model    `required:"true" json:"model"`
		DBConfig DBConfig `required:"true" json:"dbconfig"`
	}
	DBConfig struct {
		Host              string        `required:"true" json:"host"`
		Port              string        `json:"port"`
		DBName            string        `required:"true" json:"dbName"`
		UserName          string        `required:"true" json:"username"`
		Password          string        `required:"true" json:"password"`
		SSLMode           string        `required:"true" json:"sslMode"`
		Timeout           time.Duration `required:"true" json:"timeout"`
		MaxRetry          int           `required:"true" json:"maxRetry"`
		ConnectTimeout    time.Duration `required:"true" json:"connectTimeout" validate:"required,gt=0"`
		StatementTimeout  time.Duration `required:"true" json:"statementTimeout" validate:"required,gt=0"`
		MaxOpenConns      int           `required:"true" json:"maxOpenConns" validate:"required,gt=0"`
		MaxIdleConns      int           `required:"true" json:"maxIdleConns" validate:"required,gt=0"`
		ConnMaxLifetime   time.Duration `required:"true" json:"connMaxLifetime" validate:"required,gt=0"`
		ConnMaxIdleTime   time.Duration `required:"true" json:"connMaxIdleTime" validate:"required,gt=0"`
		HealthCheckPeriod time.Duration `required:"true" json:"healthCheckPeriod" validate:"required,gt=0"`
	}
)

type (
	secret struct {
		v        *viper.Viper
		Model    Model    `required:"true" json:"model"`
		DbConfig DBConfig `json:"dbconfig"`
	}
	Model struct {
		MasterKey string `json:"masterKey"`
	}
)

func (c *AppConfig) readAppConfig() {
	v := viper.New()

	v.SetTypeByDefaultValue(true)
	configsFile := filepath.Join(BaseDir(), "configs", "config.json")
	if e := os.Getenv("CONFIG_FILE_PATH"); e != "" {
		configsFile = e
	}
	v.SetConfigFile(configsFile)
	c.v = v

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	if err := v.Unmarshal(c); err != nil {
		panic(err)
	}
}

func (s *secret) readSecret() {
	v := viper.New()

	v.SetTypeByDefaultValue(true)
	secretsFile := filepath.Join(BaseDir(), "configs", "secrets.json")
	if e := os.Getenv("SECRETS_FILE_PATH"); e != "" {
		secretsFile = e
	}
	v.SetConfigFile(secretsFile)
	s.v = v

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	if err := v.Unmarshal(s); err != nil {
		panic(err)
	}
}

// BaseDir returns the directory that contains configs/config.json by first
// checking the current working directory, then its backend/ subdirectory, then
// walking up the directory tree. It defaults to the current working directory
// if no config directory can be found. This lets the app resolve config,
// migrations and the SQLite DB independently of where it is launched
// (Wails runs the dev binary from cmd/app; a built binary may be run anywhere).
func BaseDir() string {
	hasConfigs := func(dir string) bool {
		_, err := os.Stat(filepath.Join(dir, "configs", "config.json"))
		return err == nil
	}

	if exePath, err := os.Executable(); err == nil {
		if idx := strings.Index(exePath, ".app/Contents/MacOS/"); idx != -1 {
			resourcesDir := filepath.Join(exePath[:idx+len(".app/Contents")], "Resources")
			if hasConfigs(resourcesDir) {
				return resourcesDir
			}
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if hasConfigs(wd) {
		return wd
	}
	if backend := filepath.Join(wd, "backend"); hasConfigs(backend) {
		return backend
	}
	for dir := filepath.Dir(wd); dir != wd; dir = filepath.Dir(dir) {
		if hasConfigs(dir) {
			return dir
		}
	}
	return wd
}

// dataDir returns a writable, persistent directory for user data (the
// SQLite DB), separate from BaseDir since a packaged .app's Resources
// directory is read-only. On macOS this resolves to
// ~/Library/Application Support/Nexflow.
func dataDir() (string, error) {
	base, err := os.UserConfigDir() // ~/Library/Application Support on macOS
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Nexflow")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// absolutizeDBName rewrites a relative DBName in place to be relative to
// BaseDir, so the SQLite database is created in a stable location regardless of
// the process working directory.
func (c *AppConfig) absolutizeDBName() {
	if c.DBConfig.DBName == "" || filepath.IsAbs(c.DBConfig.DBName) {
		return
	}
	dir, err := dataDir()
	if err != nil {
		// fall back to previous behavior if this ever fails
		dir = BaseDir()
	}
	c.DBConfig.DBName = filepath.Join(dir, c.DBConfig.DBName)
}

func NewConfiguration() *AppConfig {
	config := &AppConfig{}
	config.readAppConfig()
	config.v.WatchConfig()
	config.v.OnConfigChange(func(in fsnotify.Event) {
		config.readAppConfig()
	})
	secrets := &secret{}
	secrets.readSecret()
	secrets.v.WatchConfig()
	secrets.v.OnConfigChange(func(in fsnotify.Event) {
		secrets.readSecret()
	})

	populateDBCredentials(secrets, config)
	config.Model = secrets.Model
	config.absolutizeDBName()

	if err := validateConfig(config); err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	return config
}

func populateDBCredentials(secret *secret, config *AppConfig) {
	if secret.DbConfig.UserName != "" {
		config.DBConfig.UserName = secret.DbConfig.UserName
	} else if dbUsername, dbUsernamePresent := os.LookupEnv("DB_USER_NAME"); dbUsernamePresent {
		config.DBConfig.UserName = dbUsername
	} else {
		slog.Error("DB_USER_NAME not found")
		os.Exit(1)
	}
	if secret.DbConfig.Password != "" {
		config.DBConfig.Password = secret.DbConfig.Password
	} else if dbPassword, dbPasswordPresent := os.LookupEnv("DB_PASSWORD"); dbPasswordPresent {
		config.DBConfig.Password = dbPassword
	} else {
		slog.Error("DB_PASSWORD not found")
		os.Exit(1)
	}
}

func validateConfig(cfg *AppConfig) error {
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		var validationErrors validator.ValidationErrors
		errors.As(err, &validationErrors)
		var errorMessages []string
		for _, e := range validationErrors {
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", e.Field(), e.Tag()))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errorMessages, "; "))
	}
	return nil
}
