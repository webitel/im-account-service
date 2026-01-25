package graphql

import (
	"context"
)

// ResolveArgs Context
type ResolveArgs struct {
	// Node to output
	Node any
	// Query of the Node
	*Query
	// Context boundary
	context.Context
}

// OutputFunc must resolve Node-* related data for output
type OutputFunc func(output *ResolveArgs) (data any, err error)
