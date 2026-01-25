package pgtypex

import (
	"cmp"
	"fmt"
	"reflect"

	"github.com/webitel/im-account-service/internal/graphql"
)

// DataField [ AS Column ] Descriptor
type DataField[T any] struct {
	Name  string   // Field / Column name
	From  []string // expression related dependency(-ies) ; [JOIN] source(s)
	Query func(ctx *FieldQuery[T]) error
	Scan  DataScanFunc[T]
	Calc  CalcField[T]
}

type CalcField[T any] struct {
	Query graphql.Fields  // OPTIONAL ; Related fields that calculation depends on ..
	Func  DataCalcFunc[T] // REQUIRED ; Function to calc related data for each row ..
}

// FieldQuery Context
type FieldQuery[T any] struct {
	// Request
	Field  *graphql.Query
	*Query // Query *Query
	// Fetch Plan
	Scan DataScanFunc[T]
	Calc DataCalcFunc[T]
}

// Dataset Field Descriptors
type DataFields[T any] map[string]DataField[T]

// JOIN Relation Descriptor
type DataJoin struct {
	Left  []string         // LEFT [table] related alias(es)
	Join  func(cte *Query) // RIGHT table ON condition(s)
	Alias string           // RIGHT [table] alias
}

// Dataset Type Descriptor
type DataType[T any] struct {
	// Select prepare NEW cte.SELECT query for this DataType[T]
	Select func(cte *Query) (plan DataScanPlan[T], err error)
	Fields DataFields[T] // map[string]DataField[TRow]
	// Deps  map[string]func(*SELECT[TRow]) // JOIN(s)
	Deps map[string]DataJoin // JOIN(s) Descriptors
	Keys []graphql.Query     // Columns MUST be always selected ; e.g. [ id ]
}

// Columns Query of the DataFields descriptors
func (dt *DataType[T]) Columns(cte *Query, req graphql.Fields) (plan DataScanPlan[T], err error) {
	var (
		ok bool
		fd DataField[T]
	)
	for _, q := range req {
		if cte.SELECT.Cols.Contains(q.Name) {
			// if slices.Contains(query.cols, name) {
			// if contains(q.cols, name) {
			continue // already ; duplicate
		}
		if fd, ok = dt.Fields[q.Name]; !ok {
			// if fd.Name != q.Name {
			return plan, fmt.Errorf(
				"dataset( %s ).column( %s ) ; not found",
				reflect.TypeFor[T]().Name(), q.Name,
			)
		}
		// AS registered
		fd.Name = cmp.Or(fd.Name, q.Name)
		// Remember as processed
		if !cte.SELECT.Cols.Append(fd.Name) {
			continue // already, once ..
		}
		// [JOIN] source(s) related
		err = dt.JoinDeps(cte, fd.From...)
		if err != nil {
			return plan, err
		}
		// [QUERY] column expression
		if fd.Query != nil {
			qt := FieldQuery[T]{
				Field: q,
				Query: cte,
				Scan:  fd.Scan,
				Calc:  fd.Calc.Func,
			}
			// [DATA] Field
			// err = fd.Query(cte, (*q))
			err = fd.Query(&qt)
			if err != nil {
				return plan, err
			}
			// plan.Scan = append(plan.Scan, fd.Scan)
			if qt.Scan != nil {
				plan.Scan = append(plan.Scan, qt.Scan)
			}
			if qt.Calc != nil {
				plan.Calc = append(plan.Calc, qt.Calc)
			}
		} else if fd.Calc.Func != nil {
			// [CALC] Field dependencies
			sub, err := dt.Columns(cte, fd.Calc.Query)
			if err != nil {
				return plan, err
			}
			sub.Calc = append(sub.Calc, fd.Calc.Func)
			plan.Append(sub)
		}
	}
	return plan, nil
}

func (dt *DataType[T]) JoinDeps(cte *Query, deps ...string) (err error) {
	n := 0
	for ; n < len(deps); n++ {
		if deps[n] == "" {
			deps = append(deps[0:n], deps[n+1:]...)
			n--
		}
		e := (n - 1)
		for ; e >= 0 && deps[e] != deps[n]; e-- {
			// lookup duplicates
		}
		if e >= 0 {
			// already seen .. skip ..
			deps = append(deps[0:n], deps[n+1:]...)
			n--
		}
	}
	if n == 0 {
		// TODO nothing
		return
	}
	var (
		stack   = make([]DataJoin, 0, n)
		resolve func(dep string) error
	)
	resolve = func(dep string) (err error) {
		if dep == "" {
			return nil // LEFT
		}
		// check if it has [once] been joined ?
		if _, ok := cte.Join[dep]; ok {
			return nil // already join[ed] ! done ..
		}
		// check if it has [plan] to be joined ?
		for _, plan := range stack {
			if plan.Alias == dep {
				return nil // plan duplicate ; skip ..
			}
		}
		// find definition
		join, ok := dt.Deps[dep]
		if !ok {
			return fmt.Errorf("join(%q) not found", dep)
		}
		// dependencies stack
		stack = append(stack, join)
		for _, sub := range join.Left {
			err = resolve(sub)
			if err != nil {
				return err
			}
		}
		// // current
		// return resolve(dep)
		// // continue
		return nil
	}
	// build dependencies plan
	for _, dep := range deps {
		err = resolve(dep)
		if err != nil {
			return err
		}
	}
	// [JOIN] dependency(-ies) plan
	for b := len(stack) - 1; b >= 0; b-- {
		dep := stack[b]
		dep.Join(cte)                           // JOIN table
		cte.Join[dep.Alias] = len(cte.Join) + 1 // number
	}
	return nil
}
