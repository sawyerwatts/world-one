package eras

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sawyerwatts/world-one/internal/common"
	"github.com/sawyerwatts/world-one/internal/db"
)

// BUG: when updating w/ stale update_time, what err does pgx return?

// TEST: test this to verify it all works
//	test containers!
//	integration test svc? or integration test db.Queries and then unit test svc?
//		or integration test both?

// TODO: add resilience
// TODO: add caching of curr era to queries, and have rollover update cache

// TODO: after this and other inlined TODOs, take a pass at the checklists

var (
	ErrWhitespaceEraName = errors.New("era name is whitespace")
	ErrDuplicateEraName  = errors.New("era name is a duplicate")
)

// Rollover is used to terminate the previous Era (if one exists) while creating
// the next Era. While this Rollover occurs, other parts of the game will likely
// be soft reset as well; because of this, the Eras cannot be rolled over before
// the actual start time of the new Era.
type Rollover struct {
	eraQueries Queries
	dbQueries  rolloverDBQueries
	slogger    *slog.Logger
}

func MakeRollover(
	eraQueries Queries,
	dbQueries rolloverDBQueries,
	slogger *slog.Logger,
) Rollover {
	return Rollover{
		eraQueries: eraQueries,
		dbQueries:  dbQueries,
		slogger:    slogger,
	}
}

type rolloverDBQueries interface {
	InsertEra(ctx context.Context, arg db.InsertEraParams) (db.Era, error)
	UpdateEra(ctx context.Context, arg db.UpdateEraParams) (db.Era, error)
}

// Exec uses sentinel errors ErrWhitespaceEraName, ErrDuplicateEraName, and
// common.ErrStaleDBInput. Exec will not commit or rollback the transaction, the
// caller is responsible for that.
func (r Rollover) Exec(
	ctx context.Context,
	now time.Time,
	newEraName string,
) (newEra db.Era, updatedEra *db.Era, _ error) {
	r.slogger.Info("Beginning the process of rolling over eras")

	isWhitespace := true
	for _, r := range newEraName {
		if !unicode.IsSpace(r) {
			isWhitespace = false
			break
		}
	}
	if isWhitespace {
		return db.Era{}, nil, ErrWhitespaceEraName
	}

	r.slogger.Info("Attempting to retrieving current era, if exists")
	currEra, err := r.eraQueries.GetCurrEra(ctx)
	if err := ctx.Err(); err != nil {
		return db.Era{}, nil, fmt.Errorf("short circuiting era rollover, context has error: %w", err)
	}
	hasCurrEra := true
	if err != nil {
		if errors.Is(err, ErrNoCurrEra) {
			hasCurrEra = false
		} else {
			return db.Era{}, nil, fmt.Errorf("era rollover failed while retrieving the current era: %w", err)
		}
	}

	if hasCurrEra {
		r.slogger.Info("There is a current era, terminating and updating database")
		currEra.EndTime = now
		updatedCurrEra, err := r.dbQueries.UpdateEra(ctx, db.UpdateEraParams{
			ID:         currEra.ID,
			Name:       currEra.Name,
			StartTime:  currEra.StartTime,
			EndTime:    currEra.EndTime,
			UpdateTime: currEra.UpdateTime,
		})
		if err := ctx.Err(); err != nil {
			return db.Era{}, nil, fmt.Errorf("short circuiting era rollover, context has error: %w", err)
		}
		if err != nil {
			if err.Error() == common.PsqlErrorMessageNoRows {
				r.slogger.Error("Failed to update the current era due to no rows returned; assuming a stale updated_time was used", slog.String("err", err.Error()))
				return db.Era{}, nil, common.ErrStaleDBInput
			}
			return db.Era{}, nil, fmt.Errorf("era rollover failed while updating the current era: %w", err)
		}
		updatedEra = &updatedCurrEra
		r.slogger.Info("Current era was saved")
	}

	r.slogger.Info("Inserting new era")
	newEra, err = r.dbQueries.InsertEra(ctx, db.InsertEraParams{
		Name:      newEraName,
		StartTime: now,
		EndTime:   common.UninitializedEndDate,
	})
	if err := ctx.Err(); err != nil {
		return db.Era{}, nil, fmt.Errorf("short circuiting era rollover, context has error: %w", err)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == common.PgErrorCodeUniqueViolation {
			r.slogger.Error("given era name is a duplicate", slog.String("givenEraName", newEraName), slog.String("err", pgErr.Error()))
			return db.Era{}, nil, ErrDuplicateEraName
		}
		return db.Era{}, nil, fmt.Errorf("era rollover failed while inserting the new era: %w", err)
	}
	r.slogger.Info("New era was saved")

	r.slogger.Info("Completing the process of rolling over eras")
	return newEra, updatedEra, nil
}
