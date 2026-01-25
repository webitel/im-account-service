package pgtypex

import "github.com/jackc/pgx/v5/pgtype"

// ScanInt64 implements pgtype.Int64Scanner
type ScanInt64 func(v pgtype.Int8) error

var _ pgtype.Int64Scanner = ScanInt64(nil)

func (fn ScanInt64) ScanInt64(v pgtype.Int8) error {
	if fn != nil {
		fn(v)
	}
	// ignore
	return nil
}
