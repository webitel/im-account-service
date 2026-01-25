package postgres

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype/zeronull"
	ua "github.com/mileusna/useragent"
	"github.com/webitel/im-account-service/internal/graphql"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/store/postgres/pgtypex"
)

var schema = struct {
	session pgtypex.DataType[model.Authorization]
	token   pgtypex.DataType[model.AccessToken]
}{}

const (
	tbl_session    = "im_account.session"
	tbl_auth_token = "im_account.session_token"
	tbl_push_token = "im_account.device"

	dep_session    = "s" // im_account.session
	dep_auth_token = "a" // im_account.session_token
	dep_push_token = "p" // im_account.device

)

func init() {

	schema.token = pgtypex.DataType[model.AccessToken]{
		Select: func(cte *pgtypex.Query) (pgtypex.DataScanPlan[model.AccessToken], error) {
			panic("TODO")
		},
		Fields: pgtypex.DataFields[model.AccessToken]{
			"id": {
				// Name: "id",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (_ error) {
					const left = dep_auth_token
					ctx.SELECT.Expr = ctx.SELECT.Expr.Column(
						pgtypex.Ident(left, "id"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					return &row.Id // NOT NULL
				},
			},
			"type": {
				// Name: "type",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (_ error) {
					const left = dep_auth_token
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(left, "type"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					return (*zeronull.Text)(&row.Type) // NULL
				},
			},
			"token": {
				// Name: "token",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (err error) {
					const left = dep_auth_token
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(left, "token"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					// return (*zeronull.Text)(&row.Token)
					return &row.Token // NOT NULL
				},
			},
			"scope": {
				// Name: "scope",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (_ error) {
					const left = dep_auth_token
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(left, "scope"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					return &row.Scope // NULL
				},
			},
			"refresh": {
				// Name: "refresh",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (_ error) {
					const left = dep_auth_token
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(left, "refresh"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					return (*zeronull.Text)(&row.Refresh) // NULL
				},
			},
			"rotated_at": {
				// Name: "rotated_at",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (_ error) {
					const left = dep_auth_token
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(left, "rotated_at"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					return (*zeronull.Timestamptz)(&row.Date) // NULL
				},
			},
			"expires_at": {
				// Name: "expires_at",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (_ error) {
					const left = dep_auth_token
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(left, "expires_at"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					return pgtypex.ScanTimestamptz(&row.Expires) // NULL
				},
			},
			"revoked_at": {
				// Name: "revoked_at",
				// From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.AccessToken]) (_ error) {
					const left = dep_auth_token
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(left, "revoked_at"),
					)
					return
				},
				Scan: func(row *model.AccessToken) any {
					return pgtypex.ScanTimestamptz(&row.Revoked) // NULL
				},
			},
			// "revoked_by": {},
		},
		Deps: map[string]pgtypex.DataJoin{},
	}

	schema.session = pgtypex.DataType[model.Authorization]{
		Select: func(cte *pgtypex.Query) (pgtypex.DataScanPlan[model.Authorization], error) {
			cte.SELECT = pgtypex.SELECT{
				// Cols: pgsqlx.names{},
				Left: dep_session,
				Join: make(map[string]any),
				Expr: pgtypex.Dialect.Select().From(
					pgtypex.Join(" ", cte.Schema.Get(tbl_session), dep_session),
				),
			}
			// MUST
			cols, _ := graphql.ParseFieldsQuery([]string{
				"dc", "id",
				"date", "name",
				"ip", "device", "device_id",
				"app_id", "contact_id",
				"metadata",
				"token",
			})
			return schema.session.Columns(cte, cols)
		},
		Fields: pgtypex.DataFields[model.Authorization]{
			"dc": {
				Name: "dc",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "dc"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return &row.Dc // NOT NULL
				},
			},
			"id": {
				Name: "id",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "id"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return &row.Id // NOT NULL
				},
			},
			"ip": {
				Name: "ip",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "ip"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return pgtypex.ScanNetIP(&row.IP) // NOT NULL
				},
			},
			"date": {
				Name: "date",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "created_at"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return (*zeronull.Timestamptz)(&row.Date)
				},
			},
			"name": {
				Name: "name",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "name"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return &row.Name // NOT NULL
				},
			},
			"app_id": {
				Name: "app_id",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "app_id"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return &row.AppId // NOT NULL
				},
			},
			"user_agent": {
				Name: "user_agent",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "user_agent"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return &row.Device.App.String // NOT NULL
				},
			},
			"device": {
				Name: "device",
				Calc: pgtypex.CalcField[model.Authorization]{
					Query: graphql.Fields{{Name: "user_agent"}},
					Func: func(row *model.Authorization) (_ error) {
						if row.Device.App.String != "" {
							row.Device.App = ua.Parse(row.Device.App.String)
						}
						return
					},
				},
			},
			"device_id": {
				Name: "device_id",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "device_id"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return &row.Device.Id // NOT NULL
				},
			},
			"contact_id": {
				Name: "contact_id",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "contact_id"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return scanContactId(&row.Contact) // NOT NULL
				},
			},
			"metadata": {
				Name: "metadata",
				From: nil, // []string{},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (_ error) {
					ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
						pgtypex.Ident(dep_session, "metadata"),
					)
					return
				},
				Scan: func(row *model.Authorization) any {
					return &row.Metadata // json.Unmarshal
				},
			},

			"token": {
				Name: "token",
				From: []string{dep_auth_token},
				Query: func(ctx *pgtypex.FieldQuery[model.Authorization]) (err error) {
					// preset: default
					if len(ctx.Field.Fields) == 0 {
						ctx.Field.Fields, err = graphql.ParseFields(
							"type,token,scope,refresh,rotated_at,expires_at,revoked_at",
							graphql.NoArgs(),
							graphql.NoNested(),
							graphql.DefaultFields(),
						)
						if err != nil {
							return err
						}
					}

					// TODO:   ROW( ..cols.. )
					plan, err := schema.token.Columns(ctx.Query, ctx.Field.Fields)
					if err != nil {
						return err
					}

					ctx.Scan = func(row *model.Authorization) any {
						return pgtypex.RecordPlanScan(plan, &row.Grant)
					}
					// const left = dep_auth_token
					// // LEFT JOIN im_account.session_token AS a
					// ctx.Query.SELECT.Expr = ctx.Query.SELECT.Expr.Column(
					// 	pgtypex.Ident(left, "token"),
					// )
					return
				},
				Scan: func(row *model.Authorization) any {
					// make
					dep := row.Grant
					if dep == nil {
						dep = &model.AccessToken{}
						row.Grant = dep
					}
					// bind
					return &dep.Token
				},
				Calc: pgtypex.CalcField[model.Authorization]{
					Func: func(row *model.Authorization) (_ error) {
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
			},
		},
		Deps: map[string]pgtypex.DataJoin{
			dep_auth_token: { // "token"
				Left: nil, // []string{},
				Join: func(cte *pgtypex.Query) {
					const alias = dep_auth_token
					cte.SELECT.Expr = cte.SELECT.Expr.JoinClause(fmt.Sprintf(
						"LEFT JOIN %[1]s %[2]s ON %[2]s.id = %[3]s.id",
						cte.Schema.Get(tbl_auth_token), alias, cte.Left,
					))
				},
				Alias: dep_auth_token,
			},
			dep_push_token: { // "push"
				Left: nil, // []string{},
				Join: func(cte *pgtypex.Query) {
					const alias = dep_push_token
					cte.SELECT.Expr = cte.SELECT.Expr.JoinClause(fmt.Sprintf(
						"LEFT JOIN %[1]s %[2]s ON %[2]s.id = %[3]s.device_id",
						cte.Schema.Get(tbl_push_token), alias, cte.Left,
					))
				},
				Alias: dep_push_token,
			},
		},
		Keys: []graphql.Query{},
	}
}
