package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgtype/zeronull"
	ua "github.com/mileusna/useragent"
	"github.com/webitel/im-account-service/infra/db/pg"
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store"
	"github.com/webitel/im-account-service/internal/store/postgres/pgtypex"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/auth/v1"
)

type SessionStore struct {
	db *pg.DB
}

func NewSessionStore(db *pg.DB) *SessionStore {
	return &SessionStore{
		db: db,
	}
}

var _ store.SessionStore = (*SessionStore)(nil)

func (c *SessionStore) Search(req store.ListSessionRequest) (*model.SessionList, error) {

	// return c.SearchV2(req)

	query, args := `
	SELECT
	-----------------------------
		a.dc, a.id
	, a.name, a.app_id
	, a.ip, a.user_agent
	, a.device_id, a.push_token
	, a.contact_id
	, a.metadata
	, a.created_at
	-----------------------------
	, z.type, z.token, z.refresh, z.scope
	, z.rotated_at, z.expires_at
	, z.revoked_at -- , z.revoked_by
	-----------------------------
	-- , c.push_token
	-----------------------------
	FROM im_account.session a
	LEFT JOIN im_account.session_token z ON a.id = z.id -- [1:1]
	-- LEFT JOIN im_account.device c ON a.device_id = c.id -- [1:1]
	`, pgx.NamedArgs{}

	// FILTER(s)
	var where []string
	if req.Dc > 0 {
		args["dc"] = req.Dc
		where = append(where, "a.dc = @dc")
	}
	if req.Id != "" {
		id, _ := uuid.Parse(req.Id)
		args["id"] = pgtype.UUID{Bytes: id, Valid: true} // even -if- not: to filter result to none with ZERO UUID
		where = append(where, "a.id = @id")
	}
	if req.Token != "" {
		args["token"] = req.Token
		where = append(where, "z.token = @token")
	}
	if req.AppId != "" {
		appId, _ := uuid.Parse(req.AppId)
		args["app_id"] = pgtype.UUID{Bytes: appId, Valid: true}
		where = append(where, "a.app_id = @app_id")
	}

	if req.DeviceId != "" {
		args["device_id"] = req.DeviceId
		where = append(where, "a.device_id = @device_id")
	}
	if req.ContactId != nil {
		args["contact_id"] = ((*ContactId)(req.ContactId))
		where = append(where, "a.contact_id = @contact_id")
	}
	if req.PushToken != nil {
		cond := "NOTNULL"
		if !*req.PushToken {
			cond = "ISNULL"
		}
		where = append(where, ("a.push_token " + cond))
	}

	// WHERE: APPLY
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	// OFFSET .. LIMIT
	if req.Page > 1 && req.Size > 0 {
		OFFSET := uint64((req.Page - 1) * req.Size)
		query += " OFFSET " + strconv.FormatUint(OFFSET, 10)
	}
	if req.Size > 0 {
		LIMIT := uint64(req.Size + 1)
		query += " LIMIT " + strconv.FormatUint(LIMIT, 10)
	}

	// PERFORM
	rows, err := c.db.Client().Query(
		req.Context, query, args,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// FETCH
	plan := pgtypex.DataScanPlan[model.Authorization]{
		Scan: []pgtypex.DataScanFunc[model.Authorization]{
			// dc
			func(row *model.Authorization) any { return &row.Dc },
			// id
			func(row *model.Authorization) any { return &row.Id },
			// name
			func(row *model.Authorization) any { return &row.Name },
			// app_id
			func(row *model.Authorization) any { return (*zeronull.Text)(&row.AppId) },
			// ip
			func(row *model.Authorization) any { return pgtypex.ScanNetIP(&row.IP) },
			// user_agent
			func(row *model.Authorization) any { return &row.Device.App.String },
			// device_id
			func(row *model.Authorization) any { return &row.Device.Id },
			// dpush_token
			func(row *model.Authorization) any {
				return pgtypex.ScanBytesFunc(func(src []byte) error {

					if len(src) == 0 {
						row.Device.Push = nil
						return nil
					}

					data := row.Device.Push
					row.Device.Push = nil
					if data == nil {
						data = &v1.PUSHSubscription{}
					}

					jsonbCodec := &protojsonCodec
					err := jsonbCodec.Unmarshal(src, data)
					if err != nil {
						return err
					}

					row.Device.Push = data
					return nil
				})
			},
			// contact_id
			func(row *model.Authorization) any { return scanContactId(&row.Contact) },
			// metadata
			func(row *model.Authorization) any { return &row.Metadata }, // json.Unmarshal
			// created_at
			func(row *model.Authorization) any { return (*zeronull.Timestamptz)(&row.Date) },
			// func(row *model.Authorization) any { dst := &row.Date; return pgtypex.ScanTimestamptz(&dst) },
			// ------------------------------------------------------------------------------------ //
			// grant.type
			func(row *model.Authorization) any {
				// init: as a first, AccessToken *struct related, column to scan ..
				row.Grant = &model.AccessToken{
					// Id: uuid.MustParse(row.Id),
				}
				return (*zeronull.Text)(&row.Grant.Type)
			},
			// grant.token
			func(row *model.Authorization) any { return (*zeronull.Text)(&row.Grant.Token) }, // NOT NULL
			// grant.refresh
			func(row *model.Authorization) any { return (*zeronull.Text)(&row.Grant.Refresh) }, // NULL
			// grant.scope
			func(row *model.Authorization) any { return &row.Grant.Scope }, // NULL
			// grant.rotated_at
			func(row *model.Authorization) any { return (*zeronull.Timestamptz)(&row.Grant.Date) }, // NOT NULL
			// grant.expires_at
			func(row *model.Authorization) any { return pgtypex.ScanTimestamptz(&row.Grant.Expires) }, // NULL
			// grant.revoked_at
			func(row *model.Authorization) any { return pgtypex.ScanTimestamptz(&row.Grant.Revoked) }, // NULL
			// ------------------------------------------------------------------------------------ //
			// // device.Push
			// func(row *model.Authorization) any { // NULL
			// 	return pgtypex.ScanBytesFunc(func(src []byte) error {
			// 		if len(src) == 0 {
			// 			row.Device.Push = nil
			// 			return nil
			// 		}
			// 		codec := &protojsonCodec
			// 		var data v1.PUSHSubscription
			// 		err := codec.Unmarshal(src, &data)
			// 		if err != nil {
			// 			return err
			// 		}
			// 		row.Device.Push = &data
			// 		return nil
			// 	})
			// },
		},
		Calc: []pgtypex.DataCalcFunc[model.Authorization]{
			// [parse]: row.Device.App.(UserAgent).String
			func(row *model.Authorization) (_ error) {
				if row.Device.App.String != "" {
					row.Device.App = ua.Parse(row.Device.App.String)
				}
				return
			},
			// [parse]: row.Grant.(*model.AccessToken)
			func(row *model.Authorization) (_ error) {
				// REQUIRED !
				if row.Grant.Token == "" {
					row.Grant = nil // sanitize
					return
				}
				// populate: session.id
				row.Grant.Id = uuid.MustParse(row.Id)
				return
			},
		},
	}

	list := pgtypex.DatasetScanner[model.Authorization]{
		Plan: plan,
		Page: &model.Dataset[model.Authorization]{
			Page: max(1, req.Page),
		},
		Size: req.Size,
	}

	err = list.ScanRows(rows)

	return list.Page, err

	// return &model.SessionList{}, nil // nothing
	// panic("not implemented")         // TODO: Implement
}

func (c *SessionStore) Delete(ctx context.Context, sessionId string) error {

	id, err := uuid.Parse(sessionId)
	if err != nil {
		// invalid [session.id] spec
		return nil
	}

	// PREPARE
	query, args := `
	WITH deleted AS
	(
		DELETE
		FROM im_account.session
		WHERE id = @id
		RETURNING id
	)
	SELECT ARRAY
	(
		SELECT id FROM deleted
	)
	`, pgx.NamedArgs{
		"id": pgtype.UUID{Bytes: id, Valid: true},
	}

	// PERFORM
	var (
		deleteIds []string
	)
	err = c.db.Client().QueryRow(
		ctx, query, args,
	).Scan(
		&deleteIds,
	)

	if err != nil {
		return err
	}

	// defer rows.Close()

	// [ OK ]
	return nil // CREATED
}

func (c *SessionStore) Create(ctx context.Context, session *model.Authorization) error {

	metadata := session.Metadata
	if metadata != nil {
		delete(metadata, "")
		if len(metadata) == 0 {
			metadata = nil
		}
	}

	query, args := `
	WITH session AS
	(
		INSERT INTO im_account.session AS w
		(
			dc, id, ip, "name"
		, app_id, device_id, user_agent
		, contact_id
		, metadata
		, created_at
		)
		VALUES
		(
			@dc, @id, @ip, @name
		, @app_id, @device_id, @user_agent
		, @contact_id
		, @metadata
		, @created_at
		)
		-- ON CONFLICT (device_id, contact_id) DO UPDATE SET --
		RETURNING *
	)
	, session_token AS
	(
		INSERT INTO im_account.session_token AS w
		(
			id, "scope"
		, "type", "token", "refresh"
		, rotated_at, expires_at
		, revoked_at -- , revoked_by
		)
		SELECT
			c.id, @scope
		, @token_type, @access_token, @refresh_token
		, @rotated_at, @expires_at
		, @revoked_at -- , @revoked_by
		FROM session c
		WHERE @access_token::text NOTNULL
		ON CONFLICT (id) DO UPDATE SET --
		  scope = @scope
		, "type" = @token_type
		, "token" = @access_token
		, "refresh" = @refresh_token
		, rotated_at = @rotated_at
		, expires_at = @expires_at
		, revoked_at = @revoked_at
		-- , revoked_by = @revoked_by
		RETURNING *
	)
	SELECT true FROM session
	`, pgx.NamedArgs{
		"dc":         session.Dc,
		"id":         session.Id, // UUID
		"ip":         pgtypex.NetIPValue(session.IP),
		"name":       session.Name,
		"app_id":     session.AppId,
		"device_id":  session.Device.Id,
		"user_agent": session.Device.App.String,
		"contact_id": (*ContactId)(session.Contact),
		"metadata":   metadata, // json.Marshal
		"created_at": pgtypex.TimestamptzValue(&session.Date),

		"scope":         session.Grant.Scope, // pgtype.FlatArray[],
		"token_type":    zeronull.Text(session.Grant.Type),
		"access_token":  zeronull.Text(session.Grant.Token),
		"refresh_token": zeronull.Text(session.Grant.Refresh),
		"rotated_at":    pgtypex.TimestamptzValue(&session.Grant.Date),
		"expires_at":    pgtypex.TimestamptzValue(session.Grant.Expires),
		"revoked_at":    pgtypex.TimestamptzValue(session.Grant.Revoked),
		// "revoked_by":    nil,
	}

	var ok bool
	err := c.db.Client().QueryRow(
		ctx, query, args,
	).Scan(&ok)

	if err != nil {
		return err
	}

	// defer rows.Close()

	// [ OK ]
	return nil // CREATED
	// panic("not implemented") // TODO: Implement
}

func (c *SessionStore) SearchV2(req store.ListSessionRequest) (*model.SessionList, error) {

	// SELECT
	cte := pgtypex.Query{
		Schema: nil,
	}

	plan, err := schema.session.Select(&cte)
	if err != nil {
		return nil, err
	}

	// FILTER
	if req.Dc > 0 {
		cte.SELECT.Expr = cte.SELECT.Expr.Where(
			pgtypex.Ident(cte.Left, "dc") + " = @dc",
		)
		cte.Params.Set("dc", req.Dc)
	}
	if req.AppId != "" {
		appId, _ := uuid.Parse(req.AppId)
		cte.SELECT.Expr = cte.SELECT.Expr.Where(
			pgtypex.Ident(cte.Left, "app_id") + " = @app_id",
		)
		cte.Params.Set("app_id", pgtype.UUID{Bytes: appId, Valid: true})
	}
	if req.Token != "" {
		const left = dep_auth_token // im_account.session_token
		err = schema.session.JoinDeps(&cte, left)
		if err != nil {
			return nil, err
		}
		cte.SELECT.Expr = cte.SELECT.Expr.Where(
			pgtypex.Ident(left, "token") + " = @token",
		)
		cte.Params.Set("token", req.Token)
	}
	if req.DeviceId != "" {
		cte.SELECT.Expr = cte.SELECT.Expr.Where(
			pgtypex.Ident(cte.Left, "device_id") + " = @device_id",
		)
		cte.Params.Set("device_id", req.DeviceId)
	}
	if req.ContactId != nil {
		// TODO
	}

	// OFFSET .. LIMIT
	if req.Page > 1 {
		if req.Size > 0 {
			OFFSET := uint64((req.Page - 1) * req.Size)
			cte.SELECT.Expr = cte.SELECT.Expr.Offset(OFFSET)
		}
	}
	if req.Size > 0 {
		LIMIT := uint64(req.Size + 1)
		cte.SELECT.Expr = cte.SELECT.Expr.Limit(LIMIT)
	}

	// PREPARE
	query, args, err := cte.ToSql()

	if err != nil {
		return nil, err
	}

	// PERFORM
	rows, err := c.db.Client().Query(
		req.Context, query, args...,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// FETCH
	var (
		res model.SessionList
		dts = pgtypex.DatasetScanner[model.Authorization]{
			Page: &res,
			Plan: plan,
		}
	)

	err = dts.ScanRows(rows)

	if err != nil {
		return nil, err
	}

	return &res, nil
}

// RegisterDevice PUSH [req.Token] for given session [req.Authorization.Id]
// If not specified try to create NEW session for ( device + contact ) authorization
// without [session.token] access grant and register device PUSH [req.Token] for it
func (c *SessionStore) RegisterDevice(req store.RegisterDeviceRequest) error {

	jsonbCodec := &protojsonCodec
	jsonbToken, err := jsonbCodec.Marshal(req.Token)
	if err != nil {
		return err
	}

	session := &req.Authorization
	// var (
	// 	createId string
	// 	updateId = session.Id
	// )
	// if updateId == "" {
	// 	// [NOTE]: Not an internal session -but- RPC authorization succeed
	// 	// - Webitel ; end-User session [access_token]
	// 	// - JWT ; app::contact issued
	// 	createId = uuid.NewString()
	// }

	// WITH device_others AS
	// (
	// 	SELECT
	// 	FROM im_account.session
	// 	LEFT JOIN UNNEST(other_uids)
	// 	WHERE device_id = @device_id
	// )
	// , others AS
	// (
	// 	UPDATE im_account.session
	// 	SET push_token = @push_token
	// 	WHERE device_id = @device_id
	// 	  AND contact_id
	// 	RETURNING true
	// )

	query, args := `
	WITH updated AS
	(
		UPDATE im_account.session SET
		  ip = coalesce(@ip, ip)   -- last address
		, user_agent = @user_agent -- last descriptor
		, push_token = @push_token -- register
		WHERE id = @id -- authorized by internal session.id
		RETURNING id -- true
	)
	, created AS
	(
		INSERT INTO im_account.session AS w
		(
			dc, ip, "name"
		, app_id, device_id, user_agent, push_token
		, contact_id
		-- , metadata
		, created_at
		)
		SELECT
			@dc, @ip, @name
		, @app_id, @device_id, @user_agent, @push_token
		, @contact_id
		-- , @metadata
		, @created_at
		WHERE @id ISNULL
		ON CONFLICT (device_id, contact_id)
		DO NOTHING -- No @session_id is given -but- such ( device + contact ) exists ; hacking ?
		-- DO UPDATE SET --
		RETURNING id -- generated
	)
	SELECT
		(SELECT true FROM updated)
	, (SELECT id FROM created)
	`, pgx.NamedArgs{
		"dc":         session.Dc,
		"id":         zeronull.Text(session.Id), // UUID ; NULL
		"ip":         pgtypex.NetIPValue(session.IP),
		"name":       model.Coalesce(session.Name, model.SessionName(&session.Device)),
		"app_id":     zeronull.Text(session.AppId), // UUID ; NULL
		"device_id":  session.Device.Id,
		"user_agent": session.Device.App.String,
		"contact_id": (*ContactId)(session.Contact),
		// "metadata":   metadata, // json.Marshal
		"created_at": pgtypex.TimestamptzValue(&session.Date),

		"push_token": json.RawMessage(jsonbToken), // protojson.Marshal
		// "other_uids": ((pgtype.FlatArray[*ContactId])(req.OtherUids)),
	}

	// PERFORM
	var ok pgtype.Bool
	err = c.db.Client().QueryRow(
		req.Context, query, args,
	).Scan(
		&ok, pgtypex.ScanTextFunc(func(src pgtype.Text) error {
			if session.Id != "" {
				return fmt.Errorf("device.register(): something went wrong")
			}
			// CREATED
			session.Id = src.String
			return nil
		}),
	)

	if err != nil {
		return err
	}

	// defer rows.Close()

	// if !ok {
	// 	// NOT Affected !
	// 	return pgx.ErrNoRows
	// }

	// [ OK ]
	return nil
}

func (c *SessionStore) UnregisterDevice(req store.UnregisterDeviceRequest) error {

	jsonbCodec := &protojsonCodec
	jsonbToken, err := jsonbCodec.Marshal(req.Token)
	if err != nil {
		return err
	}

	query, args := `
	WITH auth AS
	(
		SELECT push_token
		FROM im_account.session
		WHERE id = @session_id
	)
	, done AS
	(
		UPDATE im_account.session
		SET push_token = NULL
		WHERE id = @session_id AND push_token = @push_token
		-- RETURNING true
	)
	SELECT 
	  (SELECT NULLIF(push_token, @push_token) ISNULL FROM auth)
	-- , (SELECT count(*) FROM done)
	`, pgx.NamedArgs{
		"session_id": req.SessionId,                   // UUID
		"push_token": json.RawMessage(jsonbToken), // protojson.Marshal
	}

	// PERFORM
	var (
		ok bool
		// rows int64
	)

	err = c.db.Client().QueryRow(
		req.Context, query, args,
	).Scan(
		&ok, // &rows,
	)

	if err != nil {
		return err
	}

	// defer rows.Close()

	if !ok {
		// NOT Affected !
		return errors.BadRequest(
			errors.Message("device: invalid PUSH token"),
		)
	}

	// [ OK ]
	return nil
}
