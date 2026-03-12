package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgtype/zeronull"
	"github.com/webitel/im-account-service/internal/model"
)

type ContactId model.ContactId

func (v *ContactId) IsNull() bool {
	return v == nil || *v == ContactId{}
}

// ---------- Composite Type Support ---------- //

// -- IM Contact reference info
// CREATE TYPE im_account.refcontact AS
// (
//   --[ attribute_name data_type [ COLLATE collation ] [, ... ] ] )
//   dc  int8 -- service (internal) business (domain) identifier
// , id  uuid -- service (internal) UNIQUE subject identifier ; OPTIONAL
// , iss text -- service (external) issuer (provider) identifier ; REQUIRED
// , sub text -- service (external) subject identifier at issuer ; REQUIRED

// );

var _ pgtype.CompositeIndexGetter = (*ContactId)(nil)

// Index returns the element at fd.
func (v *ContactId) Index(fd int) any {
	switch fd {
	case 0:
		{
			// return (zeronull.Int8)(v.Dc)
			return v.Dc // int8 NOT NULL DEFAULT 0
		}
	case 1:
		{
			return &v.Id // uuid NOT NULL
		}
	case 2:
		{
			return &v.Iss // text NOT NULL
		}
	case 3:
		{
			return &v.Sub // text NOT NULL
		}
	// case 4:
	// 	{
	// 		return &v.Type
	// 	}
	}
	return fmt.Errorf("refcontact: unknown column index %d", fd)
}

var _ pgtype.CompositeIndexScanner = (*ContactId)(nil)

// ScanIndex returns a value usable as a scan target for fd.
func (v *ContactId) ScanIndex(fd int) any {
	switch fd {
	case 0:
		{
			return (*zeronull.Int8)(&v.Dc)
		}
	case 1:
		{
			// return (*zeronull.Text)(&v.Id)
			return &v.Id // uuid NOT NULL
		}
	case 2:
		{
			return &v.Iss // text NOT NULL
		}
	case 3:
		{
			return &v.Sub // text NOT NULL
		}
	// case 4:
	// 	{
	// 		return &v.Type
	// 	}
	}
	return fmt.Errorf("refcontact: unknown column index %d", fd)
}

// ScanNull sets the value to SQL NULL.
func (v *ContactId) ScanNull() error {
	// (*v) = ContactId{} // NULLify
	return fmt.Errorf("refcontact: NOT NULL")
}

/*
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

	row, err := raw.Finish()

	if err != nil {
		return dst, err
	}

	dst = pgtype.Text{
		String: string(row),
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
*/
