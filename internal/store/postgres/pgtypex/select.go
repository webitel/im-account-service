package pgtypex

// import sq "github.com/Masterminds/squirrel"

// [SELECT] Query Builder
type SELECT struct {
	// // WITH common table expression(s)
	// WITH
	// // SELECT result query
	// SELECT
	// // Named parameters
	// Params
	// // Scan plan ..
	// Plan []any

	// expr   SelectQuery    // select query expression
	// cols   Names          // select[ed] columns
	// from   string         // LEFT [table] alias
	// join   map[string]any // JOIN[ed] table [alias]
	// Schema                // map[table]view to be used instead

	Cols names          // select[ed] column names ; Used to invalidate duplicates
	Left string         // LEFT [table] alias
	Join map[string]any // JOIN [table] alias performed
	Expr SelectQuery    // SELECT Query builder
	// Scan []any          // dataset scan plan
}

func NewSelect() SELECT {
	return SELECT{
		Join: make(map[string]any),
		Expr: Dialect.Select(),
	}
}

func (q *SELECT) ToSql() (text string, args []any, err error) {
	text, args, err = q.Expr.ToSql()
	return // text, args, err
}

// func (q *SELECT) From(expr any, alias ...string) {
// 	switch from := expr.(type) {
// 	case []string:
// 		q.Query = q.Query.From(ident(from...))
// 	case string:
// 		q.Query = q.Query.From(from)
// 	default:
// 		q.Query = q.Query.FromSelect(from)
// 	}
// 	q.Left = CoalesceLast(alias...)
// }

// // LEFT [table] alias
// func (q *SELECT) LEFT() string {
// 	return q.Left
// }

func (q *SELECT) Column(expr any, alias ...string) bool {
	name := CoalesceLast(alias...)
	if name != "" && !q.Cols.Append(name) {
		// already has [once] been selected ; skip
		return false
	}
	q.Expr = q.Expr.Column(expr)
	return true
}

// type SelectExpr func(*SELECT) error

// // [SELECT]
// func Select(schema Schema, from ...string) SELECT {
// 	var table, left string
// 	switch len(from) {
// 	case 1:
// 		table = from[0]
// 	case 2:
// 		table, left = from[0], from[1]
// 	}
// 	left = cmp.Or(left, table)
// 	table = schema.Get(table)
// 	return SELECT{
// 		expr:   pgsql.Select().From(table),
// 		cols:   names{},
// 		from:   left,
// 		join:   make(map[string]any),
// 		Schema: schema,
// 	}
// }
