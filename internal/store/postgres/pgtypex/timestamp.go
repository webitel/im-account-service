package pgtypex

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/webitel/im-account-service/internal/model"
)

type ScanTimestamptzFunc func(v pgtype.Timestamptz) error

var _ pgtype.TimestamptzScanner = ScanTimestamptzFunc(nil)

func (scan ScanTimestamptzFunc) ScanTimestamptz(v pgtype.Timestamptz) error {
	if scan != nil {
		return scan(v)
	}
	// ignore
	return nil
}

func TimestamptzValue(src *time.Time) (val pgtype.Timestamptz) {
	if src == nil || src.IsZero() {
		return // pgtype.Timestamptz{Valid:false}
	}
	codec := &model.Timestamp
	return pgtype.Timestamptz{
		Time:  src.UTC().Round(codec.Precision()),
		Valid: true,
	}
}

func ScanTimestamptz(dst **time.Time) pgtype.TimestamptzScanner {
	return ScanTimestamptzFunc(func(src pgtype.Timestamptz) error {

		if !src.Valid {
			*dst = nil
			return nil
		}

		date := src.Time.Local()
		*dst = &date

		return nil
	})
}
