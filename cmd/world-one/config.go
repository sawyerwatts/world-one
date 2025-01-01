package main

import (
	"embed"
	"encoding/json"
	"errors"
	"os"
	"strings"
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

func mustReadConfig() *mainConfig {
	mainConfig := newMainConfig()

	configBytes, err := embeddedConfig.ReadFile("config.json")
	if err != nil {
		panic("embedded config filesystem failed to retrieve file: " + err.Error())
	}
	json.Unmarshal(configBytes, mainConfig)

	const pgurlEnv = "W1_PGURL"
	if pgurl, ok := os.LookupEnv(pgurlEnv); !ok {
		panic("there is not an environment variable " + pgurlEnv)
	} else {
		pgurl = strings.TrimSpace(pgurl)
		if pgurl == "" {
			panic("environment variable " + pgurlEnv + " is whitespace or empty")
		}
		mainConfig.DBConnectionString = pgurl
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
