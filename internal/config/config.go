package config

import (
	"log/slog"
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
	// Unmarshal only sees keys viper already knows, so bind each one; without
	// this, settings that come only from the environment (as in Kubernetes)
	// are ignored.
	for _, key := range []string{"USER_NAME", "PASSWORD", "SIGN_IN_URL", "CALENDAR_URL", "HEADLESS", "TIME_SLOTS"} {
		if err := viper.BindEnv(key); err != nil {
			return nil, err
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		// Expected in a container, where settings come only from the environment.
		slog.Info("no .env file, using environment only", "err", err)
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
