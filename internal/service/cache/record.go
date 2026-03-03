package cache

// Value. Record
type Value interface {
  // Set of comparable Keys
  // UNIQUE identify this Value
  Keys() []any
}

type id uint64

type record struct {
	num   id  // internal entry unique identifier
	keys  []any // comparable
	value any   // entry value
}

func (e *record) Keys() []any {
  if e != nil {
    return e.keys
  }
  return nil
}

func (e *record) Value() any {
  if e != nil {
    return e.value
  }
  return nil
}