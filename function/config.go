package main

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// config describes the available configuration
// of the running service
type config struct {
	Debug              bool
	TagKey             string `mapstructure:"tag_key"`
	TagValue           string `mapstructure:"tag_value"`
	MaxExpirationHours *int   `mapstructure:"max_expiration_hours"`
}

// Validate makes sure that the config makes sense
func (c *config) Validate() error {
	if c.TagKey == "" {
		return errors.New("Config: tag key is required")
	}
	if c.TagValue == "" {
		return errors.New("Config: tag value is required")
	}
	if &c.MaxExpirationHours == nil {
		return errors.New("Config: expiration time is required")
	}
	return nil
}

// Set the file name of the configurations file
func init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("janitor")

	defaults := map[string]interface{}{
		"debug":                true,
		"environment":          "dev",
		"max_expiration_hours": 1,
		"tag_key":              nil,
		"tag_value":            nil,
	}
	for key, value := range defaults {
		viper.SetDefault(key, value)
	}
}

// LoadConfig checks file and environment variables
func LoadConfig(logger log.FieldLogger) error {
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return errors.Wrap(err, "config load")
	}
	return errors.Wrap(cfg.Validate(), "config validate")
}
