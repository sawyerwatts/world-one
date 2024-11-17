package eras

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode"

	"github.com/sawyerwatts/world-one/internal/common"
	"github.com/sawyerwatts/world-one/internal/db"
)

// BUG: check if err is b/c of bad data (like dup name)
// BUG: when updating w/ stale update_time, what err does pgx return?

// TEST: test this to verify it all works
//	test containers!
//	integration test svc? or integration test db.Queries and then unit test svc?
//		or integration test both?

// TODO: add resilience
// TODO: add caching of curr era to queries, and have rollover update cache

// TODO: after this and other inlined TODOs, take a pass at the checklists

var ErrWhitespaceEraName = errors.New("era name is whitespace")

// Rollover is used to terminate the previous Era (if one exists) while creating
// the next Era. While this Rollover occurs, other parts of the game will likely
// be soft reset as well; because of this, the Eras cannot be rolled over before
// the actual start time of the new Era.
type Rollover struct {
	eraRepo rolloverEraRepo
	slogger slog.Logger
}

func MakeRollover(
	eraRepo rolloverEraRepo,
	slogger slog.Logger,
) Rollover {
	return Rollover{
		eraRepo: eraRepo,
		slogger: slogger,
	}
}

type rolloverEraRepo interface {
	GetCurrEra(ctx context.Context) (db.Era, error)
	InsertEra(ctx context.Context, arg db.InsertEraParams) (db.Era, error)
	UpdateEra(ctx context.Context, arg db.UpdateEraParams) (db.Era, error)
}

// Exec will return ErrWhitespaceEraName when newEraName is whitespace.
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
	currEra, err := r.eraRepo.GetCurrEra(ctx)
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
		updatedCurrEra, err := r.eraRepo.UpdateEra(ctx, db.UpdateEraParams{
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
	newEra, err = r.eraRepo.InsertEra(ctx, db.InsertEraParams{
		Name:      newEraName,
		StartTime: now,
		EndTime:   common.UninitializedEndDate,
	})
	if err := ctx.Err(); err != nil {
		return db.Era{}, nil, fmt.Errorf("short circuiting era rollover, context has error: %w", err)
	}
	if err != nil {
		return db.Era{}, nil, fmt.Errorf("era rollover failed while inserting the new era: %w", err)
	}
	r.slogger.Info("New era was saved")

	r.slogger.Info("Completing the process of rolling over eras")
	return newEra, updatedEra, nil
}
