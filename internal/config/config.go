package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Browser  BrowserConfig  `mapstructure:"browser"`
	Log      LogConfig      `mapstructure:"log"`
	Security SecurityConfig `mapstructure:"security"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout  time.Duration `mapstructure:"idleTimeout"`
}

type BrowserConfig struct {
	ExecutablePath  string        `mapstructure:"executablePath"`
	Headless        bool          `mapstructure:"headless"`
	UserDataDir     string        `mapstructure:"userDataDir"`
	ActionTimeout   time.Duration `mapstructure:"actionTimeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdownTimeout"`
	MaxSessions     int           `mapstructure:"maxSessions"`
}

type LogConfig struct {
	Level string `mapstructure:"level"` // debug, info, warn, error
}

type SecurityConfig struct {
	AllowedOrigins []string `mapstructure:"allowedOrigins"`
	ApiKey         string   `mapstructure:"apiKey"` // Example, use more robust auth
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.readTimeout", "15s")
	v.SetDefault("server.writeTimeout", "15s")
	v.SetDefault("server.idleTimeout", "60s")

	v.SetDefault("browser.executablePath", "") // Attempt auto-detect if empty
	v.SetDefault("browser.headless", true)
	v.SetDefault("browser.userDataDir", "") // Empty means temporary profile
	v.SetDefault("browser.actionTimeout", "30s")
	v.SetDefault("browser.shutdownTimeout", "10s")
	v.SetDefault("browser.maxSessions", 10) // Max concurrent browser sessions

	v.SetDefault("log.level", "info")

	v.SetDefault("security.allowedOrigins", []string{"*"}) // Be more specific in production
	v.SetDefault("security.apiKey", "")                    // Should be set via env or secure means

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.goscry")
		v.AddConfigPath("/etc/goscry")
	}

	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("GOSCRY")

	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
