package eras

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sawyerwatts/world-one/internal/common"
	"github.com/sawyerwatts/world-one/internal/db"
)

var (
	ErrWhitespaceEraName = errors.New("era name is whitespace")
	ErrDuplicateEraName  = errors.New("era name is a duplicate")
)

type rolloverDBQueries interface {
	InsertEra(ctx context.Context, arg db.InsertEraParams) (db.Era, error)
	UpdateEra(ctx context.Context, arg db.UpdateEraParams) (db.Era, error)
}

// Rollover is used to terminate the previous Era (if one exists) while creating
// the next Era. While this Rollover occurs, other parts of the game will likely
// be soft reset as well; because of this, the Eras cannot be rolled over before
// the actual start time of the new Era.
func Rollover(
	ctx context.Context,
	eraQueries Queries,
	dbQueries rolloverDBQueries,
	slogger *slog.Logger,
	now time.Time,
	newEraName string,
) (newEra db.Era, updatedEra *db.Era, _ error) {
	slogger.InfoContext(ctx, "Beginning the process of rolling over eras")

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
	newEraName = strings.TrimSpace(newEraName)

	currEra, err := eraQueries.GetCurrEra(ctx)
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
		slogger.InfoContext(ctx, "There is a current era, terminating and updating database")
		currEra.EndTime = now
		updatedCurrEra, err := dbQueries.UpdateEra(ctx, db.UpdateEraParams{
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
				slogger.ErrorContext(ctx, "Failed to update the current era due to no rows returned; assuming a stale updated_time was used", slog.String("err", err.Error()))
				return db.Era{}, nil, common.ErrStaleDBInput
			}
			return db.Era{}, nil, fmt.Errorf("era rollover failed while updating the current era: %w", err)
		}
		updatedEra = &updatedCurrEra
		slogger.InfoContext(ctx, "Current era was terminated")
	}

	slogger.InfoContext(ctx, "Inserting new era")
	newEra, err = dbQueries.InsertEra(ctx, db.InsertEraParams{
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
			slogger.ErrorContext(ctx, "given era name is a duplicate", slog.String("givenEraName", newEraName), slog.String("err", pgErr.Error()))
			return db.Era{}, nil, ErrDuplicateEraName
		}
		return db.Era{}, nil, fmt.Errorf("era rollover failed while inserting the new era: %w", err)
	}
	slogger.InfoContext(ctx, "New era was saved")

	slogger.InfoContext(ctx, "Completing the process of rolling over eras")
	return newEra, updatedEra, nil
}
