package pgtypex

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// SQL composite query statement
type Query struct {
	// // WITH common table expression(s)
	// WITH
	// // Expression of the main query
	// Expr SQLizer
	// // Named parameters
	// Args Params

	// WITH common table expression(s)
	WITH
	// SELECT of the main query
	SELECT
	// Schema map[table]view to be used instead
	Schema
	// Parameters as Named Arguments
	Params

	// Plan DataScanPlan[T]
}

var _ SQLizer = Query{}

func (q *Query) toSql() (text string, err error) {
	with, _, err := q.WITH.ToSql()
	if err != nil {
		return "", err
	}
	expr, _, err := q.SELECT.ToSql()
	if err != nil {
		return "", err
	}
	return Join("\n", with, expr), nil
	// var (
	// 	WITH   string
	// 	SELECT = qx.Expr.Suffix("") // shallowcopy
	// )
	// WITH, _, err = qx.WITH.ToSql()
	// if err != nil {
	// 	return // "", nil, err
	// }
	// if WITH != "" {
	// 	SELECT = SELECT.Prefix(WITH)
	// }
	// query, _, err = SELECT.ToSql()
	// return
}

func (q Query) ToSql() (query string, args []any, err error) {
	query, err = q.toSql()
	if err == nil && len(q.Params) > 0 {
		// // query, args, err = BindNamed(query, c.params)
		// query, args, err = pgx.NamedArgs(q.Params).RewriteQuery(
		// 	context.Background(), nil, query, nil,
		// )
		args = []any{pgx.NamedArgs(q.Params)}
	}
	return // query, args, err
}

// [C]ommon [T]able [E]xpression
type CommonTable struct {
	Name string
	Cols []string // OPTIONAL
	Expr SQLizer
}

var _ SQLizer = (*CommonTable)(nil)

func (e CommonTable) ToSql() (string, []any, error) {
	query, args, err := e.Expr.ToSql() // convertToSql(e.Source)
	if err != nil {
		return "", nil, err
	}
	var out strings.Builder
	defer out.Reset()

	out.WriteString(e.Name)
	if n := len(e.Cols); n > 0 {
		out.WriteByte('(')
		out.WriteString(e.Cols[0])
		for _, col := range e.Cols[1:] {
			out.WriteByte(',')
			out.WriteString(col)
		}
		out.WriteByte(')')
	}
	out.WriteString(" AS ")
	out.WriteByte('(')
	out.WriteString(query)
	out.WriteByte(')')
	return out.String(), args, nil
}

type WITH struct {
	Recursive bool

	expr []CommonTable  // ordered
	name map[string]int // index:name
}

func (c WITH) Has(table string) bool {
	_, ok := c.name[table]
	return ok
}

func (c *WITH) Table(name string, expr SQLizer, cols ...string) {
	// name := expr.Name
	if c.Has(name) {
		panic(fmt.Errorf("WITH %q; -- DUPLICATE", name))
	}
	e := len(c.name)
	// if cte.Recursive && id > 0 {
	// 	panic(errors.Errorf("WITH RECURSIVE %q; -- MUST be the first CTE!", name))
	// }
	if c.name == nil {
		c.name = make(map[string]int)
	}
	c.name[name] = e
	c.expr = append(c.expr, CommonTable{
		Name: name, Cols: cols, Expr: expr,
	})
}

var _ SQLizer = (*WITH)(nil)

func (c WITH) ToSql() (string, []any, error) {

	n := len(c.expr)
	if n == 0 {
		return "", nil, nil
	}

	var out strings.Builder
	defer out.Reset()

	out.WriteString("WITH ")
	if c.Recursive {
		out.WriteString("RECURSIVE ")
	}

	view := c.expr[0]
	def, _, err := view.ToSql()
	if err != nil {
		return "", nil, err
	}
	out.WriteString(def)
	for _, view := range c.expr[1:] {
		def, _, err = view.ToSql()
		if err != nil {
			return "", nil, err
		}
		out.WriteString(", ")
		out.WriteString(def)
	}
	return out.String(), nil, nil
}
