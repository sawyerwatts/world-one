package common

import "time"

// UninitializedEndDate exists because a non-nullable timestamptz in PostgreSQL
// can be mapped to a time.Time, but a nullable cannot be (at least not as
// cleanly). As such, this value in an end date indicates that the end date is
// actually unknown/undefined.
var UninitializedEndDate = time.Date(2200, time.January, 1, 0, 0, 0, 0, time.UTC)
