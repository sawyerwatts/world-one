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
	ReadTimeoutSec         int
	WriteTimeoutSec        int
	IdleTimeoutSec         int
	MaxGracefulShutdownSec int
	SlogIncludeSource      bool
	DBConnectionString     string `mapstructure:"PGURL"`
}

func readConfig() *mainConfig {
	var mainSettings *mainConfig

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

	mainSettings = newMainConfig()
	if err := v.Unmarshal(mainSettings); err != nil {
		panic("viper failed to unmarshal configs: " + err.Error())
	}
	if err := mainSettings.Validate(); err != nil {
		panic("settings failed to validate: " + err.Error())
	}

	return mainSettings
}

func newMainConfig() *mainConfig {
	return &mainConfig{
		TimeZone:               "GMT",
		Addr:                   "",
		ReadTimeoutSec:         30,
		WriteTimeoutSec:        90,
		IdleTimeoutSec:         120,
		MaxGracefulShutdownSec: 5,
		SlogIncludeSource:      false,
		DBConnectionString:     "",
	}
}

func (c *mainConfig) Validate() error {
	if c.TimeZone == "" {
		return errors.New("setting TimeZone is not initialized")
	}
	if c.Addr == "" {
		return errors.New("setting Addr is not initialized")
	}
	if c.ReadTimeoutSec < 1 {
		return errors.New("setting ReadTimeoutSec is not positive")
	}
	if c.WriteTimeoutSec < 1 {
		return errors.New("setting WriteTimeoutSec is not positive")
	}
	if c.IdleTimeoutSec < 1 {
		return errors.New("setting IdleTimeoutSec is not positive")
	}
	if c.MaxGracefulShutdownSec < 1 {
		return errors.New("setting MaxGracefulShutdownSec is not positive")
	}
	if c.DBConnectionString == "" {
		return errors.New("setting DBConnectionString is not initialized")
	}
	return nil
}
