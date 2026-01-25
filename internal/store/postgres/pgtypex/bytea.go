package pgtypex

import "github.com/jackc/pgx/v5/pgtype"

// ScanBytesFunc implements pgtype.BytesScanner
type ScanBytesFunc func(v []byte) error

var _ pgtype.BytesScanner = ScanBytesFunc(nil)

// ScanBytes receives a byte slice of driver memory
// that is only valid until the next database method call.
func (scan ScanBytesFunc) ScanBytes(v []byte) error {
	if scan != nil {
		scan(v)
	}
	// ignore
	return nil
}
