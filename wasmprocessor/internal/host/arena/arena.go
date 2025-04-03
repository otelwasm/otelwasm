package arena

import (
	"context"
	"sync"
)

type storeKeyType struct{}

var storeKey storeKeyType = struct{}{}

type store struct {
	vals []interface{}
}

var storesPool = sync.Pool{
	New: func() interface{} {
		return &store{}
	},
}

// Initialize adds an arena to the context. Any objects allocated in the host that are returned to
// wasm are added to the arena so that they are not garbage collected before the transformation
// is complete.
func Initialize(ctx context.Context) context.Context {
	s := storesPool.Get().(*store)
	s.vals = s.vals[:0]
	return context.WithValue(ctx, storeKey, s)
}

func Reset(ctx context.Context) {
	if s, ok := ctx.Value(storeKey).(*store); ok {
		storesPool.Put(s)
	} else {
		panic("BUG: store has not been initialized before calling wasm")
	}
}

func Store(ctx context.Context, ptr interface{}) uint64 {
	if s, ok := ctx.Value(storeKey).(*store); ok {
		idx := len(s.vals)
		s.vals = append(s.vals, ptr)
		return uint64(idx)
	} else {
		panic("BUG: store has not been initialized before calling wasm")
	}
}

func Load(ctx context.Context, idx uint64) interface{} {
	if s, ok := ctx.Value(storeKey).(*store); ok {
		return s.vals[idx]
	} else {
		panic("BUG: store has not been initialized before calling wasm")
	}
}
