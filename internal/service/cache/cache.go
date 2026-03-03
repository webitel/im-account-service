package cache

import (
	"fmt"
	"log/slog"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/hashicorp/golang-lru/v2/simplelru"
)

// LRU cache store.
type LRU struct {
	opts Options

	// evictCB func(node T)
	rm *record // []*Index[T]

	mu  sync.RWMutex // protects index of keys
	seq id         // last num added ; sequence
	// keys  IndexFunc[T] // extract UNIQUE keys for Value
	index map[any]id // map[key](*index).num
	table simplelru.LRUCache[id, *record]
}

func New(opts ...Option) *LRU {
  c := &LRU{
  	opts:    newOptions(opts),
  	index:   make(map[any]id),
  }
  if c.opts.TTL <= 0 {
		if size := c.opts.Size; size > 0 {
			var crit error
			c.table, crit = lru.NewWithEvict(
				size, c.evicted,
			)
			if crit != nil {
				panic(crit)
			}
		}
	}
	if c.table == nil {
		c.table = expirable.NewLRU(
			c.opts.Size, c.evicted, c.opts.TTL,
		)
	}
	return c
}

// c.mu.Lock(ed) !
func (c *LRU) evicted(_ id, row *record) {

	// c.mu.Lock()
	// defer c.mu.Unlock()

	c.delete(row.num, row.keys)

	c.opts.Log(debugLog, "[ CACHE::DEL ]",
		"", deferValue(func() slog.Value {
			return slog.GroupValue(
				// slog.Int("cache.size", c.table.Len()), // DEADLOCK(!)
				slog.Group(
					"remove",
					// "node", fmt.Sprintf("%p", node),
					slog.Uint64("num", uint64(row.num)),
					slog.Any("keys", slogKeys(row.keys)),
				),
			)
		}),
	)

	if c.opts.OnEvicted != nil {
		c.rm = row // remember node been evicted !
	}
}

func (c *LRU) delete(num id, keys []any) {

	// c.mu.Lock()
	// defer c.mu.Unlock()

	var (
		ok  bool
		rid id
	)
	for _, key := range keys {
		rid, ok = c.index[key]
		if ok && rid == num {
			delete(c.index, key)
		}
	}

}

func (c *LRU) create(num id, keys []any) {

	// c.mu.Lock()
	// defer c.mu.Unlock()

	defer func() {
		if e := recover(); e != nil {
			err := e
			_ = err
		}
	}()

	for _, key := range keys {
		c.index[key] = num
	}

}

func (c *LRU) lookup(keys []any) (rows []id) {

	// c.mu.Lock()
	// defer c.mu.Unlock()

	var (
		id id
		ok bool
		e, n int
	)

	for _, key := range keys {
		if id, ok = c.index[key]; ok {
			e, n = 0, len(rows)
			for ; e < n && rows[e] != id; e++ {
				// lookup: key(s) for the same node found ?
			}
			if e < n {
				// key node found previously !
				continue
			}
			// add to the result !
			rows = append(rows, id)
		}
	}

	return // [nums] for given [keys] found !
}

func (c *LRU) project(data any) (keys []any) {
  if c.opts.IndexKeys != nil {
    keys = c.opts.IndexKeys(data)
    if len(keys) > 0 {
      return keys
    }
  }
  if self, _ := data.(Value); self != nil {
    keys = self.Keys()
    if len(keys) > 0 {
      return keys
    }
  }
  panic("cache: add value without keys")
}

func (c *LRU) Num() int {

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.table != nil {
		return c.table.Len()
	}

	return 0
}

func (c *LRU) Add(val any) (err error) {

	var (
		node, evicted  *record
		keys, new, old []any // index keys difference !
		size           int
	)
	// outside protected section
	defer func() {

		level := debugLog
		if err != nil {
			level = slog.LevelWarn
		}
		c.opts.Log(level, "[ CACHE::SET ]",
			"", deferValue(func() slog.Value {

				if err != nil {
					return slog.GroupValue(
						slog.Int("cache.size", size),
						slog.Any("index.keys", slogKeys(keys)),
						slog.Any("error", err),
					)
				}

				args := []any{
					slog.Uint64("num", uint64(node.num)),
					slog.Any("keys", slogKeys(node.keys)),
					// "node", fmt.Sprintf("%p", node),
				}
				if n := len(new); n > 0 && n != len(node.keys) {
					args = append(
						// key(s) been appended !
						args, slog.Any("add", slogKeys(new)),
					)
				}
				if len(old) > 0 {
					args = append(
						// key(s) been removed !
						args, slog.Any("del", slogKeys(old)),
					)
				}
				// if len(args) == 2 {
				// 	// no: ( add & del ) keys difference
				// 	args = append(
				// 		// record expiry date prolonged
				// 		args, slog.String("set", "latest"),
				// 	)
				// }

				// group::index
				attrs := []slog.Attr{
					slog.Int("cache.size", size),
					slog.Group("index", args...),
				}
				// group::evicted
				if evicted != nil {
					attrs = append(attrs, slog.Group(
						"remove",
						// "node", fmt.Sprintf("%p", evicted),
						slog.Uint64("num", uint64(evicted.num)),
						slog.Any("keys", slogKeys(evicted.keys)),
					))
				}

				return slog.GroupValue(attrs...)
			}))

		// emit::onEvicted(!)
		if evicted != nil {
			c.opts.OnEvicted(evicted.value)
		}
	}()

	c.mu.Lock()
	defer c.mu.Unlock()

	size = c.table.Len()         // before
	keys = c.project(val) // for index given [set] value !
	rows := c.lookup(keys)         // node.(index).num(s) of the [keys] found !
	// // var old, new *index[V]
	// // var drop []any
	// var (
	// 	node     *Index[T]
	// 	old, new []any // index keys difference !
	// )

	// var num int // for this [node] !
	switch len(rows) {
	case 0:
		// Not Found (any) ! [ADD]
		// num = c.cache.Len() + 1
		// new = &index[V]{
		// 	num:   c.cache.Len() + 1,
		// 	keys:  keys,
		// 	value: set,
		// }
		c.seq++ // locked(!)
		node = &record{
			num:   c.seq, // c.cache.Len() + 1,
			keys:  keys,
			value: val,
		}
		// old = nil
		new = keys
	case 1:
		{
			// Found (partial) ! [EDT]
			num := rows[0]
			node, _ = c.table.Get(num) // [MUST]

			old = node.keys                     // OLD
			new = append(([]any)(nil), keys...) // copy

			node.keys = keys // NEW
			node.value = val // SET

			// [old/new] index [keys] difference
			for k, n := 0, len(old); k < n; k++ {
				for x, add := range new {
					if old[k] == add {
						new = append(new[:x], new[x+1:]...) // DO NOT [re]set !
						old = append(old[:k], old[k+1:]...) // DO NOT remove !
						n--
						k--
						break // ; new
					}
				}
			}
		}
	default:
		// some key(s) are reserved !
		err = fmt.Errorf("cache: [some] key(s) reserved")
		return
	}

	// [RE]SET ; moveToFront !
	_ = c.table.Add(node.num, node)
	size = c.table.Len() // after

	evicted, c.rm = c.rm, nil

	c.delete(node.num, old) // OLD
	c.create(node.num, new) // NEW

	// UNLOCK && fire c.evictCB(!)

	return nil
}

func (c *LRU) Get(key any) (val any) {

	var (
		ok   bool
		row *record
	)
	// UNLOCKED ; async
	defer func() {
		c.opts.Log(debugLog, "[ CACHE::GET ]",
			"lookup", deferValue(func() slog.Value {
				attrs := []slog.Attr{
					slog.Any("key", slogKeys([]any{key})),
				}
				if row != nil {
					attrs = append(attrs,
						slog.String("res", "ok"),
						slog.Uint64("num", uint64(row.num)),
					)
				} else {
					attrs = append(attrs,
						slog.String("res", "not_found"),
					)
				}
				return slog.GroupValue(attrs...)
			}),
		)
	}()

	c.mu.RLock()
	defer c.mu.RUnlock()

	num, ok := c.index[key]
	if !ok {
		return // nil
	}

	if row, ok = c.table.Get(num); ok {
		val = row.value
	}

	return // set
}

func (c *LRU) Del(val any) bool {

	var evicted *record
	// outside critical section
	defer func() {
		if evicted != nil {
			c.opts.OnEvicted(evicted.value)
		}
	}()

	c.mu.Lock()
	defer c.mu.Unlock()

	keys := c.project(val) // for index fiven [reg] value !
	rows := c.lookup(keys)          // node.(index).num(s) of the [keys] found !

	switch len(rows) {
	case 0:
		// Not Found (any) !
		return false
	case 1:
		{
			// Found (partial) ! [EDT]
			num := rows[0]
			// node, _ := c.cache.Get(num) // [MUST]
			ok := c.table.Remove(num)
			evicted, c.rm = c.rm, nil
			// c.onEvicted(here) ; [DEAD]LOCK !
			return ok
		}
	default:
		// Too much records found !
		// Do nothing !
		return false
	}
}

func (c *LRU) Range(next func(val any) bool) {

	var (
		rows []*record
	)

	c.mu.Lock()
	if c.table != nil {
		rows = c.table.Values()
	}
	c.mu.Unlock()

	i, n := 0, len(rows)
	for ; i < n && next(rows[i].value); i++ {
		// yield: next
	}
}