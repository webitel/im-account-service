package postgres

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/webitel/im-account-service/infra/db/pg"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
	"github.com/webitel/im-account-service/internal/store/postgres/pgtypex"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ store.AppStore = (*AppStore)(nil)

type AppStore struct {
	db *pg.DB
}

func NewAppStore(db *pg.DB) *AppStore {
	return &AppStore{
		db: db,
	}
}

func (c *AppStore) Search(req store.SearchAppRequest) (*model.ApplicationList, error) {

	query, args := `
	SELECT
		dc, id
	, "name", about
	, config
	FROM im_account.app
	`, pgx.NamedArgs{
		// "dc": req.Dc,
		// "id": req.Id,
	}

	var (
		where  []string
		limit  = req.Size
		offset int
	)

	if req.Page > 1 {
		if req.Size > 0 {
			offset = (req.Page - 1) * req.Size
		}
	}
	// if req.Size > 0 {
	// 	limit = req.Size + 1
	// }

	// region: filter(s)
	if req.Dc > 0 {
		where = append(where, "app.dc = @dc")
		args["dc"] = req.Dc // int8
	}
	if req.Id != "" {
		id, _ := uuid.Parse(req.Id)
		limit = 1 // ( 1 + 1 ) +1 extra
		where = append(where, "app.id = @id")
		args["id"] = pgtype.UUID{ // UUID
			Bytes: id, Valid: true, // given, but MAY be invalid, e.g. 00000000-0000-...
		}
	}
	// endregion: filter(s)

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", (limit + 1))
	}

	rows, err := c.db.Client().Query(
		req.Context, query, args,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var (
		// row *model.Application
		row *v1.Application
		res model.ApplicationList
	)

	res.Page = max(1, req.Page) // default: 1

	for rows.Next() {

		// row = &model.Application{}
		row = &v1.Application{}
		err := rows.Scan(
			// dc
			&row.Dc,
			// id
			&row.Id,
			// name
			&row.Name,
			// about
			&row.About,
			// config
			pgtypex.ScanBytesFunc(func(src []byte) error {
				enc := &protojsonCodec
				err := enc.Unmarshal(src, row)
				if err != nil {
					return err
				}
				return nil
			}),
		)

		if err != nil {
			return nil, err
		}

		rec := model.ProtoApplication(row)

		if 0 < limit && limit == len(res.Data) {
			res.Next = rec
			break // for
		}

		res.Data = append(res.Data, rec)
		// continue
	}

	return &res, nil
}

var protojsonCodec = struct {
	protojson.UnmarshalOptions
	protojson.MarshalOptions
}{
	UnmarshalOptions: protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: false,
		RecursionLimit: 0,
		Resolver:       nil,
	},
	MarshalOptions: protojson.MarshalOptions{
		Multiline:         false,
		Indent:            "",
		AllowPartial:      true,
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		EmitUnpopulated:   false,
		EmitDefaultValues: false,
		Resolver:          nil,
	},
}

func (c *AppStore) Create(req store.CreateAppRequest) (*model.Application, error) {

	src := req.App.Proto()
	enc := &protojsonCodec
	jsonb, err := enc.Marshal(src)
	if err != nil {
		return nil, err
	}

	query, args := `
	INSERT INTO im_account.app
	(
		dc, id, "name", about, config
	)
	VALUES
	(
		@dc, @id, @name, @about, @config
	)
	`, pgx.NamedArgs{
		"dc":     src.GetDc(),
		"id":     src.GetId(),
		"name":   src.GetName(),
		"about":  src.GetAbout(),
		"config": jsonb,
	}

	_, err = c.db.Client().Exec(
		req.Context, query, args,
	)

	if err != nil {
		return nil, err
	}

	return req.App, nil
}

func (c *AppStore) Update(req store.UpdateAppRequest) (*model.Application, error) {
	panic("not implemented") // TODO: Implement
}

func (c *AppStore) Revoke(req store.RevokeAppRequest) (*model.ApplicationList, error) {
	panic("not implemented") // TODO: Implement
}
