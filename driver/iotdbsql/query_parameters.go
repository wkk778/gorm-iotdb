package iotdb_go

import (
	"errors"
	"time"
)

var (
	ErrInvalidValueInNamedDateValue = errors.New("invalid value in NamedDateValue for query parameter")
	ErrUnsupportedQueryParameter    = errors.New("unsupported query parameter type")
)

func bindQueryOrAppendParameters(options *QueryOptions, query string, timezone *time.Location, args ...any) (string, error) {
	// prefer native query parameters over legacy bind if query parameters provided explicit
	if len(options.parameters) > 0 {
		return query, nil
	}

	return bind(timezone, query, args...)
}
