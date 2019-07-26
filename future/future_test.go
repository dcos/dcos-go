package future

import (
	"context"
	"errors"
	"testing"
)

func TestFutureBlockChan(t *testing.T) {
	closed := func() chan struct{} {
		c := make(chan struct{})
		close(c)
		return c
	}()
	errRandom := errors.New("some random error")
	for ti, tc := range map[string]struct {
		done   chan struct{}
		cancel <-chan struct{}
		v      interface{}
		e      error
		// -- wants:
		val interface{}
		err error
	}{
		"canceled":                   {cancel: closed},
		"done-with-val":              {done: closed, v: "1", val: "1"},
		"done-with-err":              {done: closed, e: errRandom, err: errRandom},
		"done-with-val+err":          {done: closed, v: "1", val: "1", e: errRandom, err: errRandom},
		"done-with-val,canceled":     {done: closed, v: "1", val: "1", cancel: closed},
		"done-with-err,canceled":     {done: closed, e: errRandom, err: errRandom, cancel: closed},
		"done-with-val+err,canceled": {done: closed, v: "1", val: "1", e: errRandom, err: errRandom, cancel: closed},
	} {
		t.Run(ti, func(t *testing.T) {
			for range make([]struct{}, 20) { // loop exercises tie-breaking
				f := &future{done: tc.done, v: tc.v, err: tc.e}
				v, err := f.Block(tc.cancel)

				select {
				case <-f.done:
					if tc.val != v {
						t.Fatalf("expected value %v instead of %v", tc.val, v)
					}
					if err != nil && tc.err != err {
						t.Fatalf("expected error %v instead of %v", tc.err, err)
					}
					return
				default:
				}
				select {
				case <-tc.cancel:
					if v != nil {
						t.Fatalf("unexpected value: %v", v)
					}
					if err != context.Canceled {
						t.Fatalf("unexpected error: %v", err)
					}
				default:
				}
			}
		})
	}
}

func TestFutureBlockContext(t *testing.T) {
	canceled := func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}()
	closed := func() chan struct{} {
		c := make(chan struct{})
		close(c)
		return c
	}()
	errRandom := errors.New("some random error")
	for ti, tc := range map[string]struct {
		done chan struct{}
		ctx  context.Context
		v    interface{}
		e    error
		// -- wants:
		val interface{}
		err error
	}{
		"canceled":                   {ctx: canceled},
		"done-with-val":              {done: closed, v: "1", val: "1"},
		"done-with-err":              {done: closed, e: errRandom, err: errRandom},
		"done-with-val+err":          {done: closed, v: "1", val: "1", e: errRandom, err: errRandom},
		"done-with-val,canceled":     {done: closed, v: "1", val: "1", ctx: canceled},
		"done-with-err,canceled":     {done: closed, e: errRandom, err: errRandom, ctx: canceled},
		"done-with-val+err,canceled": {done: closed, v: "1", val: "1", e: errRandom, err: errRandom, ctx: canceled},
	} {
		t.Run(ti, func(t *testing.T) {
			for range make([]struct{}, 20) { // loop exercises tie-breaking
				ctx := tc.ctx
				if ctx == nil {
					ctx = context.Background()
				}
				f := &future{done: tc.done, v: tc.v, err: tc.e}
				v, err := f.BlockContext(ctx)

				select {
				case <-f.done:
					if tc.val != v {
						t.Fatalf("expected value %v instead of %v", tc.val, v)
					}
					if err != nil && tc.err != err {
						t.Fatalf("expected error %v instead of %v", tc.err, err)
					}
					return
				default:
				}
				select {
				case <-ctx.Done():
					if v != nil {
						t.Fatalf("unexpected value: %v", v)
					}
					if err != context.Canceled {
						t.Fatalf("unexpected error: %v", err)
					}
				default:
				}
			}
		})
	}
}
