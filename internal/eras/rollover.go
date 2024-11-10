package eras

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sawyerwatts/world-one/internal/common"
	"github.com/sawyerwatts/world-one/internal/db"
)

// TODO: setup+inject logger
// TODO: add resilience
// TODO: setup Gin and make endpoint for this svc and start openapi spec
//	give openapi spec its own endpoint
// TEST: test this to verify it all works
//	test containers!
//	integration test svc? or integration test db.Queries and then unit test svc?
//		or integration test both?
// BUG: see BUG at bottom of Exec

// Rollover is used to terminate the previous Era (if one exists) while creating
// the next Era. While this Rollover occurs, other parts of the game will likely
// be soft reset as well; because of this, the Eras cannot be rolled over before
// the actual start time of the new Era.
type Rollover struct {
	queries rolloverQueries
}

func MakeRollover(queries rolloverQueries) Rollover {
	return Rollover{queries: queries}
}

type rolloverQueries interface {
	GetCurrEra(ctx context.Context) (db.Era, error)
	InsertEra(ctx context.Context, arg db.InsertEraParams) (db.Era, error)
	UpdateEra(ctx context.Context, arg db.UpdateEraParams) (db.Era, error)
}

func (rollover Rollover) Exec(
	ctx context.Context,
	now time.Time,
	newEraName string,
) (newEra db.Era, updatedEra *db.Era, _ error) {
	currEra, err := rollover.queries.GetCurrEra(ctx)
	if err := ctx.Err(); err != nil {
		return db.Era{}, nil, fmt.Errorf("short circuiting era rollover, context has error: %w", err)
	}
	hasCurrEra := true
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			hasCurrEra = false
		} else {
			return db.Era{}, nil, fmt.Errorf("era rollover failed while retrieving the current era: %w", err)
		}
	}

	if hasCurrEra {
		currEra.EndTime = now
		updatedCurrEra, err := rollover.queries.UpdateEra(ctx, db.UpdateEraParams{
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
	}

	newEra, err = rollover.queries.InsertEra(ctx, db.InsertEraParams{
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

	return newEra, updatedEra, nil
}
