package eras

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sawyerwatts/world-one/internal/common"
	"github.com/sawyerwatts/world-one/internal/db"
)

// TODO: add resilience
// TEST: test this to verify it all works
//	test containers!
//	integration test svc? or integration test db.Queries and then unit test svc?
//		or integration test both?
// BUG: see BUG at bottom of Exec
// TODO: after this and other inlined TODOs, take a pass at the checklists

// Rollover is used to terminate the previous Era (if one exists) while creating
// the next Era. While this Rollover occurs, other parts of the game will likely
// be soft reset as well; because of this, the Eras cannot be rolled over before
// the actual start time of the new Era.
type Rollover struct {
	getter    Getter
	dbQueries rolloverDBQueries
	slogger   slog.Logger
}

func MakeRollover(
	getter Getter,
	queries rolloverDBQueries,
	slogger slog.Logger,
) Rollover {
	return Rollover{
		getter:    getter,
		dbQueries: queries,
		slogger:   slogger,
	}
}

type rolloverDBQueries interface {
	InsertEra(ctx context.Context, arg db.InsertEraParams) (db.Era, error)
	UpdateEra(ctx context.Context, arg db.UpdateEraParams) (db.Era, error)
}

func (r Rollover) Exec(
	ctx context.Context,
	now time.Time,
	newEraName string,
) (newEra db.Era, updatedEra *db.Era, _ error) {
	r.slogger.Info("Beginning the process of rolling over eras")
	newEraName = strings.Trim(newEraName, " \r\n\t")
	if len(newEraName) == 0 {
		return db.Era{}, nil, fmt.Errorf("The new era's name is empty or whitespace")
		// TODO: return sentinal err? return http status/msg? make sentinal err
		// for 400 vs 500?
	}

	r.slogger.Info("Attempting to retrieving current era, if exists")
	currEra, err := r.getter.GetCurrEra(ctx)
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
		// BUG: check if err is b/c of bad data (like dup name)
		//	what to return such that 400s are obv?
		return db.Era{}, nil, fmt.Errorf("era rollover failed while inserting the new era: %w", err)
	}

	r.slogger.Info("New era was saved")
	r.slogger.Info("Completing the process of rolling over eras")
	return newEra, updatedEra, nil
}
