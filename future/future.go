package future

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type Interface interface {
	Future() Future
}

type Func func() Future

func (f Func) Future() Future {
	if f == nil {
		return nil
	}
	return f()
}

var _ = Interface(Func(nil))

type Future interface {
	Interface
	Done() <-chan struct{}
	Err() error
	Value() interface{}

	// Block blocks until the cancellation closes, or else the future completes.
	// Returns the value and error of the future (if completed).
	Block(<-chan struct{}) (interface{}, error)

	// BlockContext blocks until the context is cancelled, or else the future completes.
	// Returns the value and error of the future (if completed), or else a nil value and cancellation error.
	BlockContext(context.Context) (interface{}, error)
}

type future struct {
	done chan struct{}
	err  error
	v    interface{}
}

func (f *future) Done() <-chan struct{} { return f.done }

func (f *future) Err() (err error) {
	select {
	case <-f.done:
		err = f.err
	default:
	}
	return
}

func (f *future) Value() (v interface{}) {
	select {
	case <-f.done:
		v = f.v
	default:
	}
	return
}

func (f *future) Block(cancel <-chan struct{}) (v interface{}, err error) {
	select {
	case <-f.done:
		v, err = f.v, f.err
	case <-cancel:
		select {
		case <-f.done:
			v, err = f.v, f.err
		default:
			err = context.Canceled
		}
	}
	return
}

func (f *future) BlockContext(ctx context.Context) (v interface{}, err error) {
	select {
	case <-f.done:
		v, err = f.v, f.err
	case <-ctx.Done():
		select {
		case <-f.done:
			v, err = f.v, f.err
		default:
			// NOTE: ctx could be a Future-derived Context, in which case
			// it could be convenient to extract the completion value of
			// said future and return it here. However, we don't have a
			// requirement for that, and it also means that we prefer one
			// Future over another, which seems somewhat arbitrary.
			// It could also be surprising to the caller, whom may not
			// expect a value of the type yielded by the Context-mapped
			// Future. The safest thing to do here is to simply leave the
			// value of `v` unset.
			err = ctx.Err()
		}
	}
	return
}

func (f *future) Future() Future { return f }

type Completer interface {
	Complete(interface{}, error)
}

type CompleterFunc func(interface{}, error)

func (f CompleterFunc) Complete(v interface{}, err error) { f(v, err) }

func (f CompleterFunc) Apply(o *Options) { o.completers = append(o.completers, f) }

func (f CompleterFunc) If(b bool) (result CompleterFunc) {
	if b {
		result = f
	}
	return
}

var _ Completer = CompleterFunc(nil) // sanity check

func Do(f func()) CompleterFunc {
	return func(interface{}, error) { f() }
}

type Options struct {
	// completers are configured at promise construction-time and are invoked
	// sychronously upon promise completion. they *should not* block.
	completers []Completer
}

type Optional interface {
	Apply(*Options)
}

type Option func(*Options)

func (f Option) Apply(o *Options) { f(o) }

type Promise interface {
	Interface
	Error(error) Promise
	Value(interface{}) Promise
	Cancel() Promise

	// Complete completes this promise.
	Complete(interface{}, error)

	// OnComplete schedules a completion function that's executed after the
	// promise is completed. The given completion func never blocks the caller
	// of this method.
	OnComplete(func(interface{}, error)) Promise

	// OnPanic installs an interceptor that fires for panics in OnComplete handlers.
	OnPanic(func(Panic)) Promise

	// Completer converts this promise into a Completer. Completion of the returned
	// object completes this promise.
	Completer() CompleterFunc
}

type promise struct {
	f          Future
	complete   func(interface{}, error)
	onComplete func(func(interface{}, error))
	onPanic    func(func(Panic))
	options    Options
}

func (p promise) Future() Future                                { return p.f }
func (p promise) Error(err error) Promise                       { p.complete(nil, err); return p }
func (p promise) Value(v interface{}) Promise                   { p.complete(v, nil); return p }
func (p promise) Complete(v interface{}, err error)             { p.complete(v, err) }
func (p promise) OnComplete(f func(interface{}, error)) Promise { p.onComplete(f); return p }
func (p promise) OnPanic(f func(Panic)) Promise                 { p.onPanic(f); return p }

func (p promise) Cancel() Promise { p.complete(nil, context.Canceled); return p }

func (p promise) Completer() CompleterFunc { return p.complete }

// WithCompletion configures a synchronous completion listener for a promise, invoked at
// the time the promise is completed on the same goroutine as the promise's Complete().
func WithCompletion(c Completer) Optional {
	return Option(func(o *Options) {
		o.completers = append(o.completers, c)
	})
}

// WithCompletionF is syntactic sugar for writing completion closures.
// See WithCompletion.
func WithCompletionF(c CompleterFunc) Optional { return WithCompletion(c) }

func NewPromise(opts ...Optional) Promise {
	var (
		f = &future{
			done: make(chan struct{}),
		}
		m            sync.Mutex
		callbacks    []func(interface{}, error)
		completed    int32
		panicHandler func(Panic)
		options      Options
	)
	for _, oo := range opts {
		if oo != nil {
			oo.Apply(&options)
		}
	}
	complete := func(v interface{}, err error) {
		if atomic.LoadInt32(&completed) == 1 {
			return
		}

		m.Lock()
		if atomic.LoadInt32(&completed) == 1 {
			m.Unlock()
			return
		}

		defer func() {
			callbacks = callbacks[:0]
			close(f.done)
			// do this after channel close in case any of the callbacks
			// want to observe the "done" chan of the future: doing things
			// in this order presents a more consistent view of the future.
			atomic.StoreInt32(&completed, 1)
			m.Unlock()
		}()

		f.v = v
		f.err = err

		for i := len(options.completers); i > 0; {
			i--
			cf := options.completers[i]
			defer func() {
				success := false

				// apply panic handling to synchronous completers
				defer func() {
					if success || panicHandler == nil {
						return
					}
					x := recover()
					p := Panic{
						Recovered: x,
						Val:       v,
						Err:       err,
					}
					panicHandler(p)
				}()

				cf.Complete(v, err)
				success = true
			}()
		}

		if len(callbacks) > 0 {
			cb2 := callbacks[:]
			go func() {
				<-f.done // ensure that the promise has completed
				success := false
				defer func() {
					if success || panicHandler == nil {
						return
					}
					x := recover()
					p := Panic{
						Recovered: x,
						Val:       v,
						Err:       err,
					}
					panicHandler(p)
				}()
				for _, fn := range cb2 {
					fn(v, err)
				}
				success = true
			}()
		}
	}
	onComplete := func(fn func(interface{}, error)) {
		if fn == nil {
			return
		}
		if atomic.LoadInt32(&completed) == 1 {
			go fn(f.v, f.err)
			return
		}

		m.Lock()
		if atomic.LoadInt32(&completed) == 1 {
			m.Unlock()
			go fn(f.v, f.err)
			return
		}

		callbacks = append(callbacks, fn)
		m.Unlock()
	}
	onPanic := func(fn func(Panic)) {
		if atomic.LoadInt32(&completed) == 1 {
			return // already complete, noop
		}
		m.Lock()
		if atomic.LoadInt32(&completed) == 1 {
			m.Unlock()
			return // already complete, noop
		}

		panicHandler = fn
		m.Unlock()
	}
	return promise{
		f:          f,
		complete:   complete,
		onComplete: onComplete,
		onPanic:    onPanic,
		options:    options,
	}
}

var alwaysDone = func() (ch chan struct{}) {
	ch = make(chan struct{})
	close(ch)
	return
}()

type futureFixture struct {
	v   interface{}
	err error
}

func (futureFixture) Done() <-chan struct{}                               { return alwaysDone }
func (f futureFixture) Err() error                                        { return f.err }
func (f futureFixture) Value() interface{}                                { return f.v }
func (f futureFixture) Block(<-chan struct{}) (interface{}, error)        { return f.v, f.err }
func (f futureFixture) BlockContext(context.Context) (interface{}, error) { return f.v, f.err }
func (f futureFixture) Future() Future                                    { return f }

// Error returns a Fixture that always reports the given error.
func Error(err error) Future { return Fixture(nil, err) }

// Value returns a Fixture that always reports the given value.
func Value(v interface{}) Future { return Fixture(v, nil) }

var nilFixture = futureFixture{} // intentionally empty, always

func Nil() Future { return nilFixture }

// Fixture returns an already-completed future with the given value and error.
func Fixture(v interface{}, err error) Future { return futureFixture{v: v, err: err} }

type futureContext struct {
	f Future
}

// contextValueKind is the type for keys used to retrieve values from a context.Context.
type contextValueKind uint8

const (
	// AsContextValue may be used as a context key for retrieving the value of
	// a Future-derived Context.
	AsContextValue contextValueKind = iota
)

func (futureContext) Deadline() (_ time.Time, _ bool) { return }
func (f futureContext) Done() <-chan struct{}         { return f.f.Done() }
func (f futureContext) Value(k interface{}) (v interface{}) {
	if k != AsContextValue {
		return
	}
	select {
	case <-f.f.Done():
		v = f.f.Value()
	default:
	}
	return
}
func (f futureContext) Err() (err error) {
	// Err() must return a non-nil error value if Done() is closed.
	// If the underlying promise completes successfully then there is no
	// error value in the promise to return, so instead we return Canceled
	// since it's matches the semantics of the context package.
	select {
	case <-f.f.Done():
		err = f.f.Err()
		if err == nil {
			err = context.Canceled
		}
	default:
	}
	return
}

var _ = context.Context(&futureContext{}) // sanity check

// Context returns a context derived from the given future; it has no deadline.
func Context(f Interface) context.Context { return futureContext{f: f.Future()} }

type Merger interface {
	Merge(interface{}, error) (interface{}, error)
}

// MergeFunc performs an accumulative merge of 1 or more Future results.
type MergeFunc func(interface{}, error) (interface{}, error)

func (f MergeFunc) Merge(v interface{}, err error) (interface{}, error) { return f(v, err) }

var _ = Merger(MergeFunc(nil)) // sanity check

// Discard is a Future merge-function that discards everything and always returns (nil, nil).
func Discard() MergeFunc { return func(interface{}, error) (_ interface{}, _ error) { return } }

// FirstError is a Merger that tracks the first error its asked to Merge.
type FirstError struct{ error }

func (f *FirstError) Merge(_ interface{}, err error) (interface{}, error) {
	if f.error == nil && err != nil {
		f.error = err
	}
	return nil, f.error
}

var _ = Merger(&FirstError{}) // sanity check

// Block waits for all given futures to complete, returning the merged value of their results
// according to the specified merge-function. The given merge function SHALL NOT be invoked
// concurrently.
func Block(merge Merger, futures ...Interface) (v interface{}, err error) {
	return BlockContext(context.Background(), merge, futures...)
}

// BlockContext is like Block, but can be cancelled by the context.
func BlockContext(ctx context.Context, merge Merger, futures ...Interface) (v interface{}, err error) {
	if merge == nil {
		merge = Discard()
	}
	var (
		wg sync.WaitGroup
		m  sync.Mutex
	)
	for i := range futures {
		if futures[i] == nil {
			continue
		}
		f := futures[i].Future()
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-f.Done():
			case <-ctx.Done():
				select {
				case <-f.Done():
				default:
					m.Lock()
					defer m.Unlock()
					_, err = merge.Merge(nil, ctx.Err())
					return
				}
			}

			m.Lock()
			defer m.Unlock()
			v, err = merge.Merge(f.Value(), f.Err())
		}()
	}
	wg.Wait()
	return
}

// Join is like Block, except that it's non-blocking. The merged results are returned by Future.
func Join(merge Merger, futures ...Interface) Future {
	return JoinContext(context.Background(), merge, futures...)
}

// JoinContext is like BlockContext, except that it's non-blocking. The merged results are returned by Future.
func JoinContext(ctx context.Context, merge Merger, futures ...Interface) Future {
	p := NewPromise()
	go func() {
		v, err := BlockContext(ctx, merge, futures...)
		p.Complete(v, err)
	}()
	return p.Future()
}

// Await returns a Future that completes upon cancelation of a context.
func Await(ctx context.Context) Future {
	return ctxFuture{done: ctx.Done, err: ctx.Err}
}

type ctxFuture struct {
	done func() <-chan struct{}
	err  func() error
}

func (c ctxFuture) Future() Future        { return c }
func (c ctxFuture) Done() <-chan struct{} { return c.done() }
func (c ctxFuture) Err() error            { return c.err() }
func (c ctxFuture) Value() interface{}    { return nil }

func (c ctxFuture) Block(done <-chan struct{}) (interface{}, error) {
	select {
	case <-c.done():
		return nil, c.err()
	case <-done:
		return nil, nil
	}
}

func (c ctxFuture) BlockContext(ctx context.Context) (interface{}, error) {
	select {
	case <-c.done():
		return nil, c.err()
	case <-ctx.Done():
		// NOTE: ctx could be a Future-derived Context, in which case
		// it could be convenient to extract the completion value of
		// said future and return it here. However, we don't have a
		// requirement for that, and it also means that we prefer one
		// Future over another, which seems somewhat arbitrary.
		// It could also be surprising to the caller, whom may not
		// expect a value of the type yielded by the Context-mapped
		// Future. The safest thing to do here is to simply leave the
		// value of `v` unset.
		return nil, ctx.Err()
	}
}

// AwaitChan returns a Future that completes when the given signal chan closes.
// No data is ever expected to flow over the "done" chan, it should be closed to
// complete the returned future.
// The Err() func of the returned future returns context.Canceled once the done
// chan is closed.
// Deprecated in favor of AwaitChanNoError.
func AwaitChan(done <-chan struct{}) Future {
	return ctxFuture{
		done: func() <-chan struct{} { return done },
		err: func() (err error) {
			select {
			case <-done:
				err = context.Canceled
			default:
			}
			return
		},
	}
}

// AwaitChanNoError returns a Future that completes when the given signal chan closes.
// No data is ever expected to flow over the "done" chan, it should be closed to
// complete the returned future.
// The Err() func of the returned future returns nil, always (vs. the behavior of AwaitChan).
func AwaitChanNoError(done <-chan struct{}) Future {
	return ctxFuture{
		done: func() <-chan struct{} { return done },
		err:  func() (_ error) { return },
	}
}

// Lazy returns the value of a factory func, invoking the factory func (and caching the
// result) the first time the value is needed and returning the cached result upon subsequent
// invocations.
func Lazy(f func() interface{}) func() interface{} {
	var m sync.Once
	var val interface{}
	return func() interface{} {
		m.Do(func() {
			val = f()
		})
		return val
	}
}

// Panic captures a recovered panic value and the completion value of
// a promise whose completion handler is panicking.
type Panic struct {
	Recovered interface{}
	Val       interface{}
	Err       error
}

var ErrFlattenInvalid = errors.New("cannot flatten non-Future value")

// Flatten deferences the value of `v` as an Interface, unless v is empty (nil) or err is
// not nil. If `v` is not actually an Interface then ErrFlattenInvalid is reported by
// the returned future.
func Flatten(v interface{}, err error) Interface {
	if v == nil || err != nil {
		return Fixture(v, err)
	}
	f, ok := v.(Future)
	if !ok {
		return Fixture(v, ErrFlattenInvalid)
	}
	return f
}

// Mapper transforms the completion values of a Future.
type Mapper func(interface{}, error) Interface

// Map returns an Interface that reports the mapped value of `f` as an Interface.
// Invocations of the Interface's Future() will block until `f` is ready.
func Map(ctx context.Context, f Interface, m Mapper) Interface {
	// delayed eval allows this func to return without blocking.
	return Func(func() Future { return m(f.Future().BlockContext(ctx)).Future() })
}

// FlatMap is a convenience func that invokes Map with Flatten as the mapping func.
func FlatMap(ctx context.Context, f Interface) Interface { return Map(ctx, f, Flatten) }

// ErrorOf reduces the completion result of a future to just its error component, useful
// when the caller isn't interested in the value component of the future. Intended to be
// composed with Block and BlockContext.
func ErrorOf(_ interface{}, err error) error { return err }

// ValueOf reduces the completion result of a future to just its value component, useful
// when the caller isn't interested in the error component of the future. Intended to be
// composed with Block and BlockContext.
func ValueOf(v interface{}, _ error) interface{} { return v }

// HasCompleted returns true if the given future has reached a terminal state.
func HasCompleted(i Interface) bool {
	f := i.Future()
	select {
	case <-f.Done():
		return true
	default:
	}
	return false
}

var completedPromise = func() Promise { p := NewPromise(); p.Complete(nil, nil); return p }()

// EmptyPromise returns an already-completed promise with a nil value and error.
func EmptyPromise() Promise { return completedPromise }
