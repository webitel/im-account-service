package pgtypex

import (
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/webitel/im-account-service/internal/model"
)

// Dataset of []*T SELECT options
// type Dataset[T any] struct{}

type DataScanFunc[T any] func(row *T) any // (scan any)
type DataCalcFunc[T any] func(row *T) error

type DataScanPlan[T any] struct {
	Scan []DataScanFunc[T]
	Calc []DataCalcFunc[T]
}

func (plan *DataScanPlan[T]) Append(sub DataScanPlan[T]) {
	plan.Scan = append(plan.Scan, sub.Scan...)
	plan.Calc = append(plan.Calc, sub.Calc...)
}

// type DataRowScanner[T any] struct {
// 	Plan DataScanPlan[T]
// 	Row  *T

// 	plan []any // scan values
// }

// func (ds *DataRowScanner[T]) ScanRow(rows pgx.Rows) error {
// 	// cols := rows.FieldDescriptions()
// 	if n := len(ds.Plan.Scan); len(ds.plan) < n {
// 		ds.plan = make([]any, n)
// 	}
// 	ds.Row = new(T)
// 	for col, plan := range ds.Plan.Scan {
// 		ds.plan[col] = plan(ds.Row)
// 	}
// 	err := rows.Scan(ds.plan...)
// 	if err != nil {
// 		return err
// 	}
// 	for _, calc := range ds.Plan.Calc {
// 		err = calc(ds.Row)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

type DatasetScanner[T any] struct {
	//
	Plan DataScanPlan[T]
	Page *model.Dataset[T]
	Size int // req.Size ; LIMIT

	heap []T   // memory page
	plan []any // scan values
	row  *T    // current ROW
	r, c int   // current Dataset.Data [r]ow, [c]olumn point
}

func (ds *DatasetScanner[T]) ScanRows(rows pgx.Rows) (err error) {
	for err == nil && rows.Next() {
		err = ds.ScanRow(rows)
	}
	return // err
}

var _ pgx.RowScanner = (*DatasetScanner[any])(nil)

// ScanRow scans current ROW.
// Implements [pgx.RowScanner]
func (ds *DatasetScanner[T]) ScanRow(rows pgx.Rows) error {

	if ds.r == 0 {
		err := ds.scanPlan(rows)
		if err != nil {
			return err
		}
	}

	return ds.scanRow(rows)
}

func (c *DatasetScanner[T]) scanPlan(rows pgx.Rows) error {

	cols := len(rows.FieldDescriptions())
	if len(c.Plan.Scan) != cols {
		return fmt.Errorf("plan.Scan -vs- rows.Cols mismatch")
	}

	var (
		limit = c.Size
		page  = c.Page
	)

	if limit > 1 {
		// if size = limit - len(page); size > 1 {
		c.heap = make([]T, limit)
		if cap(page.Data) < limit {
			page.Data = make([]*T, 0, limit)
		}
		// }
	}

	// sanitize
	page.Next = nil
	if len(page.Data) > 0 {
		page.Data = page.Data[:0]
	}

	c.plan = make([]any, cols)

	// cols := len(rows.FieldDescriptions())
	// c.plan = make([]any, cols)

	// for i := range cols {
	// 	c.plan[i] = dbx.ScanFunc(c.scanValue)
	// }

	return nil
}

func (c *DatasetScanner[T]) scanRow(rows pgx.Rows) error {

	var (
		page  = c.Page
		limit = c.Size
	)

	if page.Next != nil {
		// TOO_MUCH_RECORDS
		return nil
	}

	// NEW ROW
	c.c = 0
	c.row = nil
	// allocate
	if len(c.heap) > 0 {
		c.row = &c.heap[0]
		c.heap = c.heap[1:]
	}

	if c.row == nil {
		c.row = new(T)
	}

	// bind row scan plan
	for col, bind := range c.Plan.Scan {
		if bind != nil {
			c.plan[col] = bind(c.row)
			continue
		}
		if c.r == 0 {
			// init
			c.plan[col] = DoNotScan
		}
	}

	// decode

	err := rows.Scan(c.plan...)
	if err != nil {
		return err
	}

	for _, calc := range c.Plan.Calc {
		err = calc(c.row)
		if err != nil {
			return err
		}
	}

	// result

	if 0 < limit && limit == len(page.Data) {
		page.Next = c.row // LIMIT
		return nil
	}

	page.Data = append(page.Data, c.row)
	c.r++ // APPEND

	return nil
}

// ----------------------------------------------------------------------------------- //

// Array of RECORD(s) value scanner
func (c *DatasetScanner[T]) Array() pgtype.ArraySetter {
	return pgtype.ArraySetter((*recordArrayScanner[T])(c))
}

type recordArrayScanner[T any] DatasetScanner[T]

var _ pgtype.ArraySetter = (*recordArrayScanner[any])(nil)

// inspired of [pgtype.FlatArray]

// cardinality returns the number of elements in an array of dimensions size.
func cardinality(dimensions []pgtype.ArrayDimension) int {
	if len(dimensions) == 0 {
		return 0
	}

	elementCount := int(dimensions[0].Length)
	for _, d := range dimensions[1:] {
		elementCount *= int(d.Length)
	}

	return elementCount
}

// SetDimensions prepares the value such that ScanIndex can be called for each element. This will remove any existing
// elements. dimensions may be nil to indicate a NULL array. If unable to exactly preserve dimensions SetDimensions
// may return an error or silently flatten the array dimensions.
func (c *recordArrayScanner[T]) SetDimensions(dimensions []pgtype.ArrayDimension) error {
	if dimensions == nil {
		// *v = nil
		return nil
	}

	elementCount := cardinality(dimensions)
	// *a = make(FlatArray[T], elementCount)
	if elementCount < 1 {
		// no data records
		return nil
	}

	size := elementCount
	if size > 1 {
		c.heap = make([]T, size)
		if cap(c.Page.Data) < size {
			c.Page.Data = make([]*T, 0, size)
		}
	}

	// sanitize
	c.Page.Next = nil
	if len(c.Page.Data) > 0 {
		c.Page.Data = c.Page.Data[:0]
	}

	cols := len(c.Plan.Scan)
	c.plan = make([]any, cols)

	return nil
}

// ScanIndex returns a value usable as a scan target for i. SetDimensions must be called before ScanIndex.
func (c *recordArrayScanner[T]) ScanIndex(i int) any {

	// [end] record ..
	if c.row != nil {

		if c.r == i {
			// current ; NOTE: array.ScanIndex(0) call twice ...
			return pgtype.CompositeIndexScanner((*recordPlanScanner[T])(c))
		}

		// if 0 < limit && limit == len(page.Data) {
		// 	page.Next = c.row // LIMIT
		// 	return nil
		// }

		c.Page.Data = append(c.Page.Data, c.row)
		c.r++ // APPEND
	}

	if i < 0 {
		// EOF
		return nil
	}

	if c.r != i {
		// MUST match !
		panic(fmt.Errorf("array: scan index %d != %d mismatch", c.r, i))
	}

	// [new] record ..
	c.c = 0
	c.row = nil
	// allocate
	if len(c.heap) > 0 {
		c.row = &c.heap[0]
		c.heap = c.heap[1:]
	}

	if c.row == nil {
		c.row = new(T)
	}

	// bind row scan plan
	for col, bind := range c.Plan.Scan {
		if bind != nil {
			c.plan[col] = bind(c.row)
			continue
		}
		if c.r == 0 {
			// init
			c.plan[col] = DoNotScan
		}
	}

	return pgtype.CompositeIndexScanner((*recordPlanScanner[T])(c))
}

// ScanIndexType returns a non-nil scan target of the type ScanIndex will return. This is used by
// ArrayCodec.PlanScan.
func (c *recordArrayScanner[T]) ScanIndexType() any {
	return pgtype.CompositeIndexScanner((*recordPlanScanner[T])(nil))
}

// ----------------------------------------------------------------------------------- //

func RecordPlanScan[T any](plan DataScanPlan[T], ptr **T) pgtype.CompositeIndexScanner {
	return pgtype.CompositeIndexScanner(&scanPlanRecordToCompositeIndexScanner[T]{
		plan: plan, ptr: ptr,
	})
}

type scanPlanRecordToCompositeIndexScanner[T any] struct {
	plan DataScanPlan[T]
	ptr  **T // NULL(able) target pointer
}

var _ pgtype.CompositeIndexScanner = (*scanPlanRecordToCompositeIndexScanner[any])(nil)

// ScanNull sets the value to SQL NULL.
func (c *scanPlanRecordToCompositeIndexScanner[T]) ScanNull() error {
	(*c.ptr) = nil // NULL::record
	return nil
}

// ScanIndex returns a value usable as a scan target for i.
func (c *scanPlanRecordToCompositeIndexScanner[T]) ScanIndex(i int) any {
	cols := len(c.plan.Scan)
	if cols < i {
		return fmt.Errorf("record: too much columns to scan values")
	}
	if i == 0 { // && (*c.ptr) == nil {
		(*c.ptr) = new(T)
	}
	scanPlan := c.plan.Scan[i]
	return scanPlan(*c.ptr)
}

// func (c *RecordScanner[T]) ScanBytes(src []byte) {
// 	raw := pgtype.NewCompositeBinaryScanner()
// }

// func (c *RecordScanner[T]) ScanText(src pgtype.Text) {
// 	raw := pgtype.NewCompositeTextScanner()
// }

// RECORD(s) value scanner
func (c *DatasetScanner[T]) Record() pgtype.CompositeIndexScanner {
	return pgtype.CompositeIndexScanner((*recordPlanScanner[T])(c))
}

type recordPlanScanner[T any] DatasetScanner[T]

var _ pgtype.CompositeIndexScanner = (*recordPlanScanner[any])(nil)

// ScanNull sets the value to SQL NULL.
func (c *recordPlanScanner[T]) ScanNull() error {
	// TODO: NULL record !
	c.Page.Data = append(c.Page.Data, nil)
	c.r++
	return nil
}

// ScanIndex returns a value usable as a scan target for i.
func (c *recordPlanScanner[T]) ScanIndex(i int) any {

	// // cols := len(c.Plan.Scan)

	// switch i {
	// case -1: // EOF
	// 	{
	// 		if c.row != nil {
	// 			c.Page.Data = append(c.Page.Data, c.row)
	// 			// record [end]
	// 			c.r++
	// 			c.c = 0
	// 			c.row = nil
	// 		}
	// 		// EOF
	// 		return nil
	// 	}
	// case 0: // next record !
	// 	{
	// 		// this [end] record ..
	// 		if c.row != nil {
	// 			c.Page.Data = append(c.Page.Data, c.row)
	// 			c.r++ // [end] record ..
	// 		}
	// 		// next [new] record ..
	// 		c.c = 0
	// 		c.row = nil
	// 		// allocate
	// 		if len(c.heap) > 0 {
	// 			c.row = &c.heap[0]
	// 			c.heap = c.heap[1:]
	// 		}

	// 		if c.row == nil {
	// 			c.row = new(T)
	// 		}

	// 		// bind row scan plan
	// 		for col, fieldPlan := range c.Plan.Scan {
	// 			if fieldPlan != nil {
	// 				c.scan[col] = fieldPlan(c.row)
	// 				continue
	// 			}
	// 			if c.r == 0 {
	// 				// init
	// 				c.scan[col] = doNotScan
	// 			}
	// 		}
	// 	}
	// 	// case (cols - 1): // last value ?
	// 	// 	{

	// 	// 	}
	// 	// default:
	// 	// 	{

	// 	// 	}
	// }

	// ( i < len(c.scan) )
	return c.plan[i]
}

// ----------------------------------------------------------------------------------- //
