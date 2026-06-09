package redislock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrLockNotAcquired = errors.New("redis lock not acquired")
	ErrLockNotHeld     = errors.New("redis lock not held")
	ErrLockReleased    = errors.New("redis lock already released")
)

type LockError struct {
	Op    string
	Key   string
	Cause error
	Msg   string
}

func (e *LockError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Cause != nil {
		return fmt.Sprintf("redis lock %s failed for key %q: %v", e.Op, e.Key, e.Cause)
	}
	return fmt.Sprintf("redis lock %s failed for key %q", e.Op, e.Key)
}

func (e *LockError) Unwrap() error {
	return e.Cause
}

type Backend interface {
	Acquire(ctx context.Context, key, token string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, token string) error
	Refresh(ctx context.Context, key, token string, ttl time.Duration) error
}

type RedisBackend struct {
	client *goredis.Client
}

var releaseScript = goredis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

var refreshScript = goredis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0
`)

func NewRedisBackend(client *goredis.Client) *RedisBackend {
	return &RedisBackend{client: client}
}

func (b *RedisBackend) Acquire(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	return b.client.SetNX(ctx, key, token, ttl).Result()
}

func (b *RedisBackend) Release(ctx context.Context, key, token string) error {
	deleted, err := releaseScript.Run(ctx, b.client, []string{key}, token).Int64()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrLockNotHeld
	}
	return nil
}

func (b *RedisBackend) Refresh(ctx context.Context, key, token string, ttl time.Duration) error {
	updated, err := refreshScript.Run(ctx, b.client, []string{key}, token, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if updated == 0 {
		return ErrLockNotHeld
	}
	return nil
}

type RetryStrategy interface {
	Next(attempt int) (time.Duration, bool)
}

type retryStrategyFunc func(attempt int) (time.Duration, bool)

func (f retryStrategyFunc) Next(attempt int) (time.Duration, bool) {
	return f(attempt)
}

type Option func(*lockOptions)

type lockOptions struct {
	retryStrategy RetryStrategy
	maxRetries    int
	backoff       time.Duration
	autoRefresh   bool
}

const (
	defaultMaxRetries = 20
	defaultBackoff    = 50 * time.Millisecond
)

func WithRetryStrategy(strategy RetryStrategy) Option {
	return func(o *lockOptions) {
		o.retryStrategy = strategy
	}
}

func WithMaxRetries(maxRetries int) Option {
	return func(o *lockOptions) {
		o.maxRetries = maxRetries
	}
}

func WithBackoff(backoff time.Duration) Option {
	return func(o *lockOptions) {
		o.backoff = backoff
	}
}

func WithAutoRefresh() Option {
	return func(o *lockOptions) {
		o.autoRefresh = true
	}
}

func FixedBackoff(backoff time.Duration, maxRetries int) RetryStrategy {
	if backoff <= 0 {
		backoff = defaultBackoff
	}
	return retryStrategyFunc(func(attempt int) (time.Duration, bool) {
		if maxRetries < 0 {
			return 0, false
		}
		if attempt > maxRetries {
			return 0, false
		}
		return backoff, true
	})
}

func LinearBackoff(step time.Duration, maxRetries int) RetryStrategy {
	if step <= 0 {
		step = defaultBackoff
	}
	return retryStrategyFunc(func(attempt int) (time.Duration, bool) {
		if maxRetries < 0 {
			return 0, false
		}
		if attempt > maxRetries {
			return 0, false
		}
		return time.Duration(attempt) * step, true
	})
}

func defaultOptions() lockOptions {
	return lockOptions{
		maxRetries: defaultMaxRetries,
		backoff:    defaultBackoff,
	}
}

func (o lockOptions) retry() RetryStrategy {
	if o.retryStrategy != nil {
		return o.retryStrategy
	}
	return FixedBackoff(o.backoff, o.maxRetries)
}

type Manager struct {
	backend Backend
	queue   *localKeyQueue
}

func NewManager(backend Backend) *Manager {
	return &Manager{
		backend: backend,
		queue:   newLocalKeyQueue(),
	}
}

type Lock struct {
	backend      Backend
	key          string
	token        string
	ttl          time.Duration
	releaseLocal func()
	watchdogStop chan struct{}
	finalizeOnce sync.Once

	mu       sync.Mutex
	released bool
}

func (l *Lock) Key() string { return l.key }
func (l *Lock) Token() string { return l.token }

func (l *Lock) Release(ctx context.Context) error {
	l.mu.Lock()
	if l.released {
		l.mu.Unlock()
		return ErrLockReleased
	}
	l.mu.Unlock()

	err := l.backend.Release(ctx, l.key, l.token)
	if err != nil {
		if errors.Is(err, ErrLockNotHeld) {
			l.finalize()
			return &LockError{Op: "release", Key: l.key, Cause: ErrLockNotHeld}
		}
		return &LockError{Op: "release", Key: l.key, Cause: err}
	}

	l.finalize()
	return nil
}

func (l *Lock) Refresh(ctx context.Context, ttl time.Duration) error {
	l.mu.Lock()
	if l.released {
		l.mu.Unlock()
		return ErrLockReleased
	}
	l.mu.Unlock()

	err := l.backend.Refresh(ctx, l.key, l.token, ttl)
	if err != nil {
		if errors.Is(err, ErrLockNotHeld) {
			l.finalize()
			return &LockError{Op: "refresh", Key: l.key, Cause: ErrLockNotHeld}
		}
		return &LockError{Op: "refresh", Key: l.key, Cause: err}
	}
	return nil
}

func (l *Lock) finalize() {
	l.finalizeOnce.Do(func() {
		l.mu.Lock()
		l.released = true
		l.mu.Unlock()
		close(l.watchdogStop)
		if l.releaseLocal != nil {
			l.releaseLocal()
		}
	})
}

func (l *Lock) startWatchdog(ctx context.Context) {
	if l.ttl <= 0 {
		return
	}

	ctx = context.WithoutCancel(ctx)
	interval := l.ttl / 2
	if interval <= 0 {
		interval = l.ttl
	}
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-l.watchdogStop:
				return
			case <-ticker.C:
				refreshCtx, cancel := context.WithTimeout(ctx, interval)
				_ = l.Refresh(refreshCtx, l.ttl)
				cancel()
			}
		}
	}()
}

func (m *Manager) TryLock(ctx context.Context, key string, ttl time.Duration) (*Lock, bool, error) {
	if key == "" {
		return nil, false, &LockError{Op: "tryLock", Key: key, Msg: "redis lock key is required"}
	}
	if ttl <= 0 {
		return nil, false, &LockError{Op: "tryLock", Key: key, Msg: "redis lock ttl must be greater than zero"}
	}

	releaseLocal, ok := m.queue.tryAcquire(key)
	if !ok {
		return nil, false, nil
	}

	token, err := newToken()
	if err != nil {
		releaseLocal()
		return nil, false, &LockError{Op: "tryLock", Key: key, Cause: err}
	}

	acquired, err := m.backend.Acquire(ctx, key, token, ttl)
	if err != nil {
		releaseLocal()
		return nil, false, &LockError{Op: "tryLock", Key: key, Cause: err}
	}
	if !acquired {
		releaseLocal()
		return nil, false, nil
	}

	return newLock(m.backend, key, token, ttl, releaseLocal), true, nil
}

func (m *Manager) Acquire(ctx context.Context, key string, ttl time.Duration, opts ...Option) (*Lock, error) {
	if key == "" {
		return nil, &LockError{Op: "acquire", Key: key, Msg: "redis lock key is required"}
	}
	if ttl <= 0 {
		return nil, &LockError{Op: "acquire", Key: key, Msg: "redis lock ttl must be greater than zero"}
	}

	options := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	releaseLocal, err := m.queue.acquire(ctx, key)
	if err != nil {
		return nil, &LockError{Op: "acquire", Key: key, Cause: err}
	}

	ownedByLock := false
	defer func() {
		if !ownedByLock {
			releaseLocal()
		}
	}()

	token, err := newToken()
	if err != nil {
		return nil, &LockError{Op: "acquire", Key: key, Cause: err}
	}

	strategy := options.retry()
	attempt := 0

	for {
		acquired, acquireErr := m.backend.Acquire(ctx, key, token, ttl)
		if acquireErr != nil {
			return nil, &LockError{Op: "acquire", Key: key, Cause: acquireErr}
		}
		if acquired {
			lock := newLock(m.backend, key, token, ttl, releaseLocal)
			if options.autoRefresh {
				lock.startWatchdog(ctx)
			}
			ownedByLock = true
			return lock, nil
		}

		attempt++
		delay, ok := strategy.Next(attempt)
		if !ok {
			return nil, &LockError{Op: "acquire", Key: key, Cause: ErrLockNotAcquired}
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, &LockError{Op: "acquire", Key: key, Cause: ctx.Err()}
		case <-timer.C:
		}
	}
}

func (m *Manager) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error, opts ...Option) error {
	if fn == nil {
		return &LockError{Op: "withLock", Key: key, Msg: "redis lock callback is required"}
	}

	lock, err := m.Acquire(ctx, key, ttl, opts...)
	if err != nil {
		return err
	}

	callbackErr := fn(ctx)
	releaseErr := lock.Release(ctx)
	if callbackErr != nil {
		return callbackErr
	}
	return releaseErr
}

func newLock(backend Backend, key, token string, ttl time.Duration, releaseLocal func()) *Lock {
	return &Lock{
		backend:      backend,
		key:          key,
		token:        token,
		ttl:          ttl,
		releaseLocal: releaseLocal,
		watchdogStop: make(chan struct{}),
	}
}

func newToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

type localKeyQueue struct {
	mu   sync.Mutex
	keys map[string]*keyQueueState
}

type keyQueueState struct {
	held    bool
	waiters []chan struct{}
}

func newLocalKeyQueue() *localKeyQueue {
	return &localKeyQueue{keys: make(map[string]*keyQueueState)}
}

func (q *localKeyQueue) tryAcquire(key string) (func(), bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	state := q.keys[key]
	if state == nil {
		state = &keyQueueState{}
		q.keys[key] = state
	}
	if state.held {
		return nil, false
	}

	state.held = true
	return q.releaseFunc(key), true
}

func (q *localKeyQueue) acquire(ctx context.Context, key string) (func(), error) {
	q.mu.Lock()
	state := q.keys[key]
	if state == nil {
		state = &keyQueueState{}
		q.keys[key] = state
	}

	if !state.held {
		state.held = true
		q.mu.Unlock()
		return q.releaseFunc(key), nil
	}

	waiter := make(chan struct{})
	state.waiters = append(state.waiters, waiter)
	q.mu.Unlock()

	select {
	case <-waiter:
		return q.releaseFunc(key), nil
	case <-ctx.Done():
		q.removeWaiter(key, waiter)
		return nil, ctx.Err()
	}
}

func (q *localKeyQueue) removeWaiter(key string, waiter chan struct{}) {
	q.mu.Lock()
	defer q.mu.Unlock()

	state := q.keys[key]
	if state == nil {
		return
	}

	for index, queued := range state.waiters {
		if queued == waiter {
			state.waiters = append(state.waiters[:index], state.waiters[index+1:]...)
			break
		}
	}

	if !state.held && len(state.waiters) == 0 {
		delete(q.keys, key)
	}
}

func (q *localKeyQueue) releaseFunc(key string) func() {
	var once sync.Once
	return func() {
		once.Do(func() { q.release(key) })
	}
}

func (q *localKeyQueue) release(key string) {
	q.mu.Lock()
	state := q.keys[key]
	if state == nil {
		q.mu.Unlock()
		return
	}

	if len(state.waiters) == 0 {
		delete(q.keys, key)
		q.mu.Unlock()
		return
	}

	next := state.waiters[0]
	state.waiters = state.waiters[1:]
	q.mu.Unlock()
	close(next)
}
