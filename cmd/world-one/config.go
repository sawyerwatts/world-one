package main

import (
	"bytes"
	"embed"
	"errors"

	"github.com/spf13/viper"
)

//go:embed config.json
var embeddedConfig embed.FS

type mainConfig struct {
	TimeZone               string
	Addr                   string
	ReadHeaderTimeoutMS    int
	ReadTimeoutMS          int
	IdleTimeoutMS          int
	RequestTimeoutMS       int
	MaxGracefulShutdownSec int
	SlogIncludeSource      bool
	DBConnectionString     string `mapstructure:"PGURL"`
	WebsiteDir             string
}

func readConfig() *mainConfig {
	var mainConfig *mainConfig

	v := viper.New()
	v.SetEnvPrefix("W1")
	v.BindEnv("PGURL")

	configBytes, err := embeddedConfig.ReadFile("config.json")
	if err != nil {
		panic("embedded config filesystem failed to retrieve file: " + err.Error())
	}
	v.SetConfigType("json")
	if err := v.ReadConfig(bytes.NewReader(configBytes)); err != nil {
		panic("viper failed to read configs: " + err.Error())
	}

	mainConfig = newMainConfig()
	if err := v.Unmarshal(mainConfig); err != nil {
		panic("viper failed to unmarshal configs: " + err.Error())
	}
	if err := mainConfig.Validate(); err != nil {
		panic("config failed to validate: " + err.Error())
	}

	return mainConfig
}

func newMainConfig() *mainConfig {
	return &mainConfig{
		TimeZone:               "GMT",
		Addr:                   "",
		ReadHeaderTimeoutMS:    500,
		ReadTimeoutMS:          500,
		IdleTimeoutMS:          30_000,
		RequestTimeoutMS:       10_000,
		MaxGracefulShutdownSec: 5,
		SlogIncludeSource:      false,
		DBConnectionString:     "",
	}
}

func (c *mainConfig) Validate() error {
	if c.TimeZone == "" {
		return errors.New("config TimeZone is not initialized")
	}
	if c.Addr == "" {
		return errors.New("config Addr is not initialized")
	}
	if c.ReadHeaderTimeoutMS < 1 {
		return errors.New("config ReadHeaderTimeoutMS is not positive")
	}
	if c.ReadTimeoutMS < 1 {
		return errors.New("config ReadTimeoutMS is not positive")
	}
	if c.IdleTimeoutMS < 1 {
		return errors.New("config IdleTimeoutMS is not positive")
	}
	if c.RequestTimeoutMS < 1 {
		return errors.New("config RequestTimeoutMS is not positive")
	}
	if c.MaxGracefulShutdownSec < 1 {
		return errors.New("config MaxGracefulShutdownSec is not positive")
	}
	if c.DBConnectionString == "" {
		return errors.New("config DBConnectionString is not initialized")
	}
	if c.WebsiteDir == "" {
		return errors.New("config WebsiteDir is not initialized")
	}
	return nil
}
