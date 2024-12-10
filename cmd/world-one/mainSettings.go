package main

import "errors"

type mainSettings struct {
	TimeZone               string
	Addr                   string
	ReadTimeoutSec         int
	WriteTimeoutSec        int
	IdleTimeoutSec         int
	MaxGracefulShutdownSec int
	SlogIncludeSource      bool
	DBConnectionString     string `mapstructure:"PGURL"`
}

func newMainSettings() *mainSettings {
	return &mainSettings{
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

func (s *mainSettings) Validate() error {
	if s.TimeZone == "" {
		return errors.New("TimeZone is not initialized")
	}
	if s.Addr == "" {
		return errors.New("Addr is not initialized")
	}
	if s.ReadTimeoutSec < 1 {
		return errors.New("ReadTimeoutSec is not positive")
	}
	if s.WriteTimeoutSec < 1 {
		return errors.New("WriteTimeoutSec is not positive")
	}
	if s.IdleTimeoutSec < 1 {
		return errors.New("IdleTimeoutSec is not positive")
	}
	if s.MaxGracefulShutdownSec < 1 {
		return errors.New("MaxGracefulShutdownSec is not positive")
	}
	if s.DBConnectionString == "" {
		return errors.New("DBConnectionString is not initialized")
	}
	return nil
}
