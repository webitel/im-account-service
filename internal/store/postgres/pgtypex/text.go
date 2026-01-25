package pgtypex

import "github.com/jackc/pgx/v5/pgtype"

// ScanBytesFunc implements pgtype.BytesScanner
type ScanTextFunc func(src pgtype.Text) error

var _ pgtype.TextScanner = ScanTextFunc(nil)

// ScanBytes receives a byte slice of driver memory
// that is only valid until the next database method call.
func (scan ScanTextFunc) ScanText(src pgtype.Text) error {
	if scan != nil {
		scan(src)
	}
	// ignore
	return nil
}
