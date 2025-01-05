package eras

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/sawyerwatts/world-one/internal/db"
)

// TODO: impl caching

var ErrNoCurrEra = errors.New("there is no current era, the game is misconfigured")

type Queries struct {
	dbQueries queriesDBQueries
	slogger   *slog.Logger
}

func MakeQueries(
	dbQueries queriesDBQueries,
	slogger *slog.Logger,
) Queries {
	return Queries{
		dbQueries: dbQueries,
		slogger:   slogger,
	}
}

type queriesDBQueries interface {
	GetCurrEra(ctx context.Context) (db.Era, error)
	GetEras(ctx context.Context) ([]db.Era, error)
}

// GetCurrEra can return ErrNoCurrEra.
func (q Queries) GetCurrEra(ctx context.Context) (db.Era, error) {
	q.slogger.InfoContext(ctx, "Retrieving current era")
	currEra, err := q.dbQueries.GetCurrEra(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Era{}, ErrNoCurrEra
		}
		return db.Era{}, fmt.Errorf("era queries failed to retrieve current era: %w", err)
	}
	q.slogger.InfoContext(ctx, "Retrieved current era")
	return currEra, nil
}

func (q Queries) GetEras(ctx context.Context) ([]db.Era, error) {
	q.slogger.InfoContext(ctx, "Retrieving eras")
	allEras, err := q.dbQueries.GetEras(ctx)
	if err != nil {
		return nil, fmt.Errorf("era queries failed to retrieve all eras: %w", err)
	}
	q.slogger.InfoContext(ctx, "Retrieved eras")
	return allEras, nil
}
