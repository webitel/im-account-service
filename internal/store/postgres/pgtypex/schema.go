package pgtypex

// Schema map[table]name
type Schema map[string]string

// Map schema [table] to common [view] table expression
func (m *Schema) Map(table, view string) bool {
	h := (*m)
	if h == nil {
		h = make(Schema)
		(*m) = h
	}
	if cte, ok := h[table]; ok {
		return cte == view
	}
	h[table] = view
	return true
}

// Get schema [table] view
func (m Schema) Get(table string) string {
	view, ok := m[table]
	if ok && view != "" {
		// wrapped
		return view
	}
	// default
	return table
}
