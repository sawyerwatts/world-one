package eras

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/sawyerwatts/world-one/internal/db"
)

// TODO: add caching of curr era

var ErrNoCurrEra = errors.New("there is no current era, the game is misconfigured")

type EraRepo struct {
	dbQueries *db.Queries
	slogger   *slog.Logger
}

func MakeEraRepo(
	dbQueries *db.Queries,
	slogger *slog.Logger,
) EraRepo {
	return EraRepo{
		dbQueries: dbQueries,
		slogger:   slogger,
	}
}

// GetCurrEra may return ErrNoCurrEra
func (er EraRepo) GetCurrEra(ctx context.Context) (db.Era, error) {
	er.slogger.Info("Retrieving current era")
	currEra, err := er.dbQueries.GetCurrEra(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			er.slogger.Error(ErrNoCurrEra.Error())
			return db.Era{}, ErrNoCurrEra
		}
		return db.Era{}, fmt.Errorf("era getter failed to retrieve current era: %w", err)
	}
	er.slogger.Info("Retrieved current era")
	return currEra, nil
}

func (er EraRepo) GetEras(ctx context.Context) ([]db.Era, error) {
	er.slogger.Info("Retrieving eras")
	allEras, err := er.dbQueries.GetEras(ctx)
	if err != nil {
		return nil, fmt.Errorf("era getter failed to retrieve all eras: %w", err)
	}
	er.slogger.Info("Retrieved eras")
	return allEras, nil
}
