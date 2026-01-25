package pgtypex

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	JSONBCodec = pgtype.JSONBCodec{
		Marshal:   json.Marshal,
		Unmarshal: json.Unmarshal,
	}
)

func JSONBValue(src any) (json.RawMessage, error) {
	raw, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}
