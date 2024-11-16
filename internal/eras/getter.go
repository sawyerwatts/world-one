package eras

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/sawyerwatts/world-one/internal/db"
)

var ErrNoCurrEra = errors.New("There is no current era, the game is misconfigured")

// TODO: add caching of curr era
//	curr era cache svc? repo??

type Getter struct {
	dbQueries getterDBQueries
	slogger   *slog.Logger
}

func MakeGetter(
	dbQueries getterDBQueries,
	slogger *slog.Logger,
) Getter {
	return Getter{
		dbQueries: dbQueries,
		slogger:   slogger,
	}
}

type getterDBQueries interface {
	GetEras(ctx context.Context) ([]db.Era, error)
	GetCurrEra(ctx context.Context) (db.Era, error)
}

// GetCurrEra may return ErrNoCurrEra
func (g Getter) GetCurrEra(ctx context.Context) (db.Era, error) {
	g.slogger.Info("Retrieving current era")
	currEra, err := g.dbQueries.GetCurrEra(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			g.slogger.Error(ErrNoCurrEra.Error())
			return db.Era{}, ErrNoCurrEra
		}
		return db.Era{}, fmt.Errorf("era getter failed to retrieve current era: %w", err)
	}
	g.slogger.Info("Retrieved current era")
	return currEra, nil
}

func (g Getter) GetEras(ctx context.Context) ([]db.Era, error) {
	g.slogger.Info("Retrieving eras")
	allEras, err := g.dbQueries.GetEras(ctx)
	if err != nil {
		return nil, fmt.Errorf("era getter failed to retrieve all eras: %w", err)
	}
	g.slogger.Info("Retrieved eras")
	return allEras, nil
}
