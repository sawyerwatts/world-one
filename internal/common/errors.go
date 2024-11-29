package common

import "errors"

var ErrStaleDBInput = errors.New("stale input data was given to a SQL query")
