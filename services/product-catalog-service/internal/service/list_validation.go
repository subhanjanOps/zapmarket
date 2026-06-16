package service

import (
	"fmt"
	"strings"

	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

// validateSortField rejects a sort_by value that isn't in allowed, instead
// of silently falling back to a default — an unrecognized sort field is a
// client error (typo, stale API doc), not something to paper over.
func validateSortField(sortBy string, allowed map[string]bool) error {
	if sortBy == "" {
		return nil
	}
	if !allowed[sortBy] {
		fields := make([]string, 0, len(allowed))
		for f := range allowed {
			if f != "" {
				fields = append(fields, f)
			}
		}
		return pkgerrors.NewValidation(
			"INVALID_SORT_FIELD",
			fmt.Sprintf("sort_by must be one of: %s", strings.Join(fields, ", ")),
		)
	}
	return nil
}

// capPageSize enforces domain.MaxPageSize and applies domain.DefaultPageSize
// when limit is unset, so a client can't force an unbounded table scan.
func capPageSize(limit, defaultSize, maxSize int) int {
	if limit <= 0 {
		return defaultSize
	}
	if limit > maxSize {
		return maxSize
	}
	return limit
}
