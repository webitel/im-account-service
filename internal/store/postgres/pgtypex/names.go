package pgtypex

import "slices"

// Names represents a SET of unique names
type names []string

func (vs names) Index(v string) int {
	// return slices.IndexFunc(vs)
	return slices.Index(vs, v)
}

func (vs names) Contains(v string) bool {
	return vs.Index(v) >= 0
}

func (vs *names) Append(v string) bool {
	ns := (*vs)
	if ns == nil {
		(*vs) = []string{v}
		return true
	}
	if !ns.Contains(v) {
		(*vs) = append(ns, v)
		return true
	}
	return false
}
