package redislock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeBackend struct {
	mu sync.Mutex

	values map[string]string

	acquireDelay time.Duration
	refreshCount int

	concurrentAcquire int
	maxConcurrent     int
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		values: make(map[string]string),
	}
}

func (f *fakeBackend) Acquire(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	f.mu.Lock()
	f.concurrentAcquire++
	if f.concurrentAcquire > f.maxConcurrent {
		f.maxConcurrent = f.concurrentAcquire
	}
	delay := f.acquireDelay
	f.mu.Unlock()

	if delay > 0 {
		select {
		case <-ctx.Done():
			f.mu.Lock()
			f.concurrentAcquire--
			f.mu.Unlock()
			return false, ctx.Err()
		case <-time.After(delay):
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.concurrentAcquire--

	if _, exists := f.values[key]; exists {
		return false, nil
	}
	f.values[key] = token
	return true, nil
}

func (f *fakeBackend) Release(ctx context.Context, key, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, ok := f.values[key]
	if !ok || value != token {
		return ErrLockNotHeld
	}
	delete(f.values, key)
	return nil
}

func (f *fakeBackend) Refresh(ctx context.Context, key, token string, ttl time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	value, ok := f.values[key]
	if !ok || value != token {
		return ErrLockNotHeld
	}
	f.refreshCount++
	return nil
}

func (f *fakeBackend) set(key, token string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key] = token
}

func (f *fakeBackend) get(key string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.values[key]
}

func TestManagerTryLock(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	manager := NewManager(backend)

	lock, ok, err := manager.TryLock(context.Background(), "lock:user:1", time.Second)
	if err != nil {
		t.Fatalf("TryLock returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected first TryLock to acquire lock")
	}
	if lock == nil {
		t.Fatal("expected non-nil lock")
	}

	secondLock, secondOK, secondErr := manager.TryLock(context.Background(), "lock:user:1", time.Second)
	if secondErr != nil {
		t.Fatalf("second TryLock returned error: %v", secondErr)
	}
	if secondOK {
		t.Fatal("expected second TryLock to fail while lock is held")
	}
	if secondLock != nil {
		t.Fatal("expected nil lock when TryLock fails")
	}

	if err := lock.Release(context.Background()); err != nil {
		t.Fatalf("Release returned error: %v", err)
	}
}

func TestManagerAcquireRetrySuccess(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	managerOne := NewManager(backend)
	managerTwo := NewManager(backend)

	firstLock, ok, err := managerOne.TryLock(context.Background(), "lock:user:2", time.Second)
	if err != nil || !ok {
		t.Fatalf("setup TryLock failed: ok=%v err=%v", ok, err)
	}

	go func() {
		time.Sleep(60 * time.Millisecond)
		_ = firstLock.Release(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	secondLock, acquireErr := managerTwo.Acquire(
		ctx,
		"lock:user:2",
		time.Second,
		WithBackoff(10*time.Millisecond),
		WithMaxRetries(20),
	)
	if acquireErr != nil {
		t.Fatalf("Acquire returned error: %v", acquireErr)
	}
	if secondLock == nil {
		t.Fatal("expected non-nil second lock")
	}
	_ = secondLock.Release(context.Background())
}

func TestManagerAcquireContextTimeout(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	managerOne := NewManager(backend)
	managerTwo := NewManager(backend)

	firstLock, ok, err := managerOne.TryLock(context.Background(), "lock:user:3", time.Second)
	if err != nil || !ok {
		t.Fatalf("setup TryLock failed: ok=%v err=%v", ok, err)
	}
	defer func() {
		_ = firstLock.Release(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	_, acquireErr := managerTwo.Acquire(
		ctx,
		"lock:user:3",
		time.Second,
		WithBackoff(15*time.Millisecond),
		WithMaxRetries(100),
	)
	if acquireErr == nil {
		t.Fatal("expected Acquire to fail on context timeout")
	}
	if !errors.Is(acquireErr, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", acquireErr)
	}
}

func TestLockReleaseOnlyDeletesOwnedToken(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	manager := NewManager(backend)

	lock, ok, err := manager.TryLock(context.Background(), "lock:user:4", time.Second)
	if err != nil || !ok {
		t.Fatalf("setup TryLock failed: ok=%v err=%v", ok, err)
	}

	backend.set("lock:user:4", "other-token")

	releaseErr := lock.Release(context.Background())
	if !errors.Is(releaseErr, ErrLockNotHeld) {
		t.Fatalf("expected ErrLockNotHeld, got %v", releaseErr)
	}
	if got := backend.get("lock:user:4"); got != "other-token" {
		t.Fatalf("expected lock token to stay unchanged, got %q", got)
	}
}

func TestLockRefreshOnlyRefreshesOwnedToken(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	manager := NewManager(backend)

	lock, ok, err := manager.TryLock(context.Background(), "lock:user:5", time.Second)
	if err != nil || !ok {
		t.Fatalf("setup TryLock failed: ok=%v err=%v", ok, err)
	}

	backend.set("lock:user:5", "other-token")

	refreshErr := lock.Refresh(context.Background(), time.Second)
	if !errors.Is(refreshErr, ErrLockNotHeld) {
		t.Fatalf("expected ErrLockNotHeld, got %v", refreshErr)
	}
}

func TestManagerWithLock(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	manager := NewManager(backend)

	err := manager.WithLock(context.Background(), "lock:user:6", time.Second, func(ctx context.Context) error {
		if got := backend.get("lock:user:6"); got == "" {
			t.Fatal("expected lock to exist during callback")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock returned error: %v", err)
	}
	if got := backend.get("lock:user:6"); got != "" {
		t.Fatalf("expected lock to be released, got %q", got)
	}

	callbackErr := errors.New("callback failed")
	err = manager.WithLock(context.Background(), "lock:user:7", time.Second, func(ctx context.Context) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("expected callback error, got %v", err)
	}
	if got := backend.get("lock:user:7"); got != "" {
		t.Fatalf("expected lock to be released after callback error, got %q", got)
	}
}

func TestManagerAutoRefresh(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	manager := NewManager(backend)

	err := manager.WithLock(
		context.Background(),
		"lock:user:8",
		40*time.Millisecond,
		func(ctx context.Context) error {
			time.Sleep(120 * time.Millisecond)
			return nil
		},
		WithAutoRefresh(),
	)
	if err != nil {
		t.Fatalf("WithLock returned error: %v", err)
	}
	if backend.refreshCount == 0 {
		t.Fatal("expected auto refresh to refresh at least once")
	}

	backendWithoutRefresh := newFakeBackend()
	managerWithoutRefresh := NewManager(backendWithoutRefresh)
	err = managerWithoutRefresh.WithLock(
		context.Background(),
		"lock:user:9",
		40*time.Millisecond,
		func(ctx context.Context) error {
			time.Sleep(120 * time.Millisecond)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("WithLock without auto refresh returned error: %v", err)
	}
	if backendWithoutRefresh.refreshCount != 0 {
		t.Fatalf("expected no refresh without auto refresh, got %d", backendWithoutRefresh.refreshCount)
	}
}

func TestManagerLocalQueueSerializesSameKey(t *testing.T) {
	t.Parallel()

	backend := newFakeBackend()
	backend.acquireDelay = 25 * time.Millisecond
	manager := NewManager(backend)

	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock, err := manager.Acquire(
				context.Background(),
				"lock:user:10",
				time.Second,
				WithBackoff(10*time.Millisecond),
				WithMaxRetries(20),
			)
			if err != nil {
				t.Errorf("Acquire returned error: %v", err)
				return
			}
			time.Sleep(20 * time.Millisecond)
			if err := lock.Release(context.Background()); err != nil {
				t.Errorf("Release returned error: %v", err)
			}
		}()
	}

	wg.Wait()

	if backend.maxConcurrent > 1 {
		t.Fatalf("expected same-key acquire attempts to be serialized, maxConcurrent=%d", backend.maxConcurrent)
	}
}
