package pgtypex

import (
	"database/sql"
	"database/sql/driver"
)

// ScanFunc implements a database/sql.Scanner
type ScanFunc func(src any) error

var _ sql.Scanner = ScanFunc(nil)

// Scan assigns a value from a database driver.
//
// The src value will be of one of the following types:
//
//	int64
//	float64
//	bool
//	[]byte
//	string
//	time.Time
//	nil - for NULL values
//
// An error should be returned if the value cannot be stored
// without loss of information.
//
// Reference types such as []byte are only valid until the next call to Scan
// and should not be retained. Their underlying memory is owned by the driver.
// If retention is necessary, copy their values before the next call to Scan.
func (scan ScanFunc) Scan(src any) error {
	// implemented ?
	if scan != nil {
		return scan(src)
	}
	// ignore
	return nil
}

// sql.Scanner that does nothing
var DoNotScan = ScanFunc(nil)

// ValueFunc implements a database/sql/driver.Valuer
type ValueFunc func() (any, error)

var _ driver.Valuer = ValueFunc(nil)

// Value returns a database/sql/driver.Value.
// Value must not panic.
func (eval ValueFunc) Value() (driver.Value, error) {
	// implemented ?
	if eval != nil {
		return eval()
	}
	// NULL
	return nil, nil
}
