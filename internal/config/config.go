package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type UserConfig struct {
	UserName string `mapstructure:"USER_NAME"`
	Password string `mapstructure:"PASSWORD"`
}

type SiteConfig struct {
	SignInURL   string `mapstructure:"SIGN_IN_URL"`
	CalendarURL string `mapstructure:"CALENDAR_URL"`
}

type RunConfig struct {
	Headless bool `mapstructure:"HEADLESS"`
}

type BookingConfig struct {
	TimeSlotsStr string `mapstructure:"TIME_SLOTS"`
	TimeSlots    []string
}

type Config struct {
	UserConfig    `mapstructure:",squash"` // "squash" handles the embedding
	SiteConfig    `mapstructure:",squash"`
	RunConfig     `mapstructure:",squash"`
	BookingConfig `mapstructure:",squash"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// the file is missing in production (using system Envs),
		fmt.Printf("Warning: .env file not found or unreadable: %v\n", err)
	}

	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Parse time slots from comma-separated string
	if cfg.BookingConfig.TimeSlotsStr != "" {
		cfg.BookingConfig.TimeSlots = strings.Split(strings.TrimSpace(cfg.BookingConfig.TimeSlotsStr), ",")
		for i := range cfg.BookingConfig.TimeSlots {
			cfg.BookingConfig.TimeSlots[i] = strings.TrimSpace(cfg.BookingConfig.TimeSlots[i])
		}
	}

	return &cfg, nil
}
