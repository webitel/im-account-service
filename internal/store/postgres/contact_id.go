package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/webitel/im-account-service/infra/db/pg"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store/postgres/pgtypex"
)

type ContactId model.ContactId

func (v *ContactId) IsNull() bool {
	return v == nil || *v == ContactId{}
}

func (v *ContactId) TextValue() (dst pgtype.Text, err error) {

	if v.IsNull() {
		return // pgtype.Text{Valid: false}, nil
	}

	// rec := pgtype.CompositeFields{
	// 	&v.Dc,
	// 	&v.Id,
	// 	&v.Iss,
	// 	&v.Sub,
	// }

	pgtypes := pg.Default().TypeMap()
	// pgtypes.Encode(pgtype.RecordOID, pgtype.TextFormatCode, rec, nil)
	// plan := pgtypes.PlanEncode(pgtype.RecordOID, pgtype.TextFormatCode, rec)
	// plan.Encode(rec, nil)

	raw := pgtype.NewCompositeTextBuilder(pgtypes, nil)

	raw.AppendValue(pgtype.Int8OID, v.Dc)
	raw.AppendValue(pgtype.TextOID, v.Id)
	raw.AppendValue(pgtype.TextOID, v.Iss)
	raw.AppendValue(pgtype.TextOID, v.Sub)

	text, err := raw.Finish()

	if err != nil {
		return dst, err
	}

	dst = pgtype.Text{
		String: string(text),
		Valid:  true,
	}
	return
}

func (v *ContactId) ScanText(src pgtype.Text) error {

	if !src.Valid {
		if !v.IsNull() {
			*v = ContactId{}
		}
		return nil
	}

	// rec := pgtype.CompositeFields{
	// 	&v.Dc,
	// 	&v.Id,
	// 	&v.Iss,
	// 	&v.Sub,
	// }

	pgtypes := pg.Default().TypeMap()
	// err := pgtypes.Scan(pgtype.RecordOID, pgtype.TextFormatCode, []byte(src.String), rec)
	// if err != nil {
	// 	return err
	// }

	// return nil

	// // conn, err := pg.Default().Client().Acquire(context.Background())
	// // conn.Conn().TypeMap()

	raw := pgtype.NewCompositeTextScanner(nil, []byte(src.String))
	for _, col := range []any{
		&v.Dc,
		&v.Id,
		&v.Iss,
		&v.Sub,
	} {

		if !raw.Next() {
			return fmt.Errorf("scan %q into *ContactId ; too few values", src.String)
		}
		rtype, _ := pgtypes.TypeForValue(col)
		err := rtype.Codec.PlanScan(pgtypes, rtype.OID, pgtype.TextFormatCode, col).Scan(raw.Bytes(), col)
		if err != nil {
			return fmt.Errorf("scan %q into *ContactId ; %v", src.String, err)
		}
	}

	if raw.Next() {
		return fmt.Errorf("scan %q into *ContactId ; too many values", src.String)
	}

	return raw.Err()
}

func scanContactId(ref **model.ContactId) any {
	// pgtype.TextScanner
	return pgtypex.ScanTextFunc(func(src pgtype.Text) error {
		switch src.String {
		case "", "()":
			{
				(*ref) = nil
				return nil
			}
		}
		var dst ContactId
		err := dst.ScanText(src)
		if err != nil {
			return err
		}
		if dst.IsNull() {
			(*ref) = nil
			return nil
		}
		(*ref) = (*model.ContactId)(&dst)
		return nil
	})
}
