package invocation

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey int

const invocationKey ctxKey = 0

// ContextWithInvocation stores the run invocation uuid on a context so the
// session store can group all events of a cycle under one run.
func ContextWithInvocation(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, invocationKey, id)
}

// LookupInvocation returns the run invocation uuid stored on the context.
func LookupInvocation(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(invocationKey).(uuid.UUID)
	return id, ok
}
