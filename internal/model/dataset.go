package model

import (
	"net/http"

	"github.com/webitel/im-account-service/internal/errors"
)

type Offset any

// Dataset of *[T] records
type Dataset[T any] struct {
	// List of dataset records
	Data []*T
	// This page offset, if specified
	Page Offset
	// Next page offset, if available
	Next Offset
	// Total records count, beyond this page
	Total int
}

var (
	ErrTooManyRecords = errors.New(
		errors.Code(http.StatusConflict),
		errors.Status("CONFLICT"),
		errors.Message("too many records"),
	)
	ErrNoRecordsFound = errors.NotFound(
		errors.Message("no records found"),
	)
)

// Get ensures that given dataset [page] contains exact one result record.
func Get[T any](list *Dataset[T], err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	if list == nil {
		// Not Found
		return nil, nil
	}
	size := len(list.Data)
	if list.Next != nil || size > 1 {
		return nil, ErrTooManyRecords
	}
	var row *T
	if size == 1 {
		row = list.Data[0]
	}
	// if obj == nil {
	// 	return nil, ErrNotFound
	// }
	return row, nil
}
