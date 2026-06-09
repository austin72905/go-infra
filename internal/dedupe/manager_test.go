package dedupe

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeBackend struct {
	store map[string]time.Time
	err   error
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		store: make(map[string]time.Time),
	}
}

func (f *fakeBackend) TryMark(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	expireAt, ok := f.store[key]
	if ok && time.Now().Before(expireAt) {
		return false, nil
	}
	f.store[key] = time.Now().Add(ttl)
	return true, nil
}

func (f *fakeBackend) Marked(ctx context.Context, key string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	expireAt, ok := f.store[key]
	if !ok {
		return false, nil
	}
	if time.Now().After(expireAt) {
		delete(f.store, key)
		return false, nil
	}
	return true, nil
}

func (f *fakeBackend) Clear(ctx context.Context, key string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.store, key)
	return nil
}

func TestManagerTryMark(t *testing.T) {
	t.Parallel()

	manager := NewManager(newFakeBackend())

	ok, err := manager.TryMark(context.Background(), "event:1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("TryMark returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected first TryMark to succeed")
	}

	ok, err = manager.TryMark(context.Background(), "event:1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("second TryMark returned error: %v", err)
	}
	if ok {
		t.Fatal("expected second TryMark to fail while TTL is active")
	}

	time.Sleep(70 * time.Millisecond)

	ok, err = manager.TryMark(context.Background(), "event:1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("third TryMark returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected TryMark to succeed after TTL expires")
	}
}

func TestManagerMarkedAndClear(t *testing.T) {
	t.Parallel()

	manager := NewManager(newFakeBackend())

	marked, err := manager.Marked(context.Background(), "event:2")
	if err != nil {
		t.Fatalf("Marked returned error: %v", err)
	}
	if marked {
		t.Fatal("expected key to be initially unmarked")
	}

	ok, err := manager.TryMark(context.Background(), "event:2", time.Second)
	if err != nil || !ok {
		t.Fatalf("TryMark failed: ok=%v err=%v", ok, err)
	}

	marked, err = manager.Marked(context.Background(), "event:2")
	if err != nil {
		t.Fatalf("Marked after TryMark returned error: %v", err)
	}
	if !marked {
		t.Fatal("expected key to be marked")
	}

	if err := manager.Clear(context.Background(), "event:2"); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}

	marked, err = manager.Marked(context.Background(), "event:2")
	if err != nil {
		t.Fatalf("Marked after Clear returned error: %v", err)
	}
	if marked {
		t.Fatal("expected key to be cleared")
	}
}

func TestManagerValidationErrors(t *testing.T) {
	t.Parallel()

	manager := NewManager(newFakeBackend())

	_, err := manager.TryMark(context.Background(), "", time.Second)
	if !errors.Is(err, ErrDedupeKeyRequired) {
		t.Fatalf("expected ErrDedupeKeyRequired, got %v", err)
	}

	_, err = manager.TryMark(context.Background(), "event:3", 0)
	if !errors.Is(err, ErrDedupeTTLRequired) {
		t.Fatalf("expected ErrDedupeTTLRequired, got %v", err)
	}

	_, err = manager.Marked(context.Background(), "")
	if !errors.Is(err, ErrDedupeKeyRequired) {
		t.Fatalf("expected ErrDedupeKeyRequired, got %v", err)
	}

	err = manager.Clear(context.Background(), "")
	if !errors.Is(err, ErrDedupeKeyRequired) {
		t.Fatalf("expected ErrDedupeKeyRequired, got %v", err)
	}
}

func TestManagerBackendErrors(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("redis unavailable")
	manager := NewManager(&fakeBackend{
		store: make(map[string]time.Time),
		err:   backendErr,
	})

	_, err := manager.TryMark(context.Background(), "event:4", time.Second)
	if !errors.Is(err, backendErr) {
		t.Fatalf("expected backend error from TryMark, got %v", err)
	}

	_, err = manager.Marked(context.Background(), "event:4")
	if !errors.Is(err, backendErr) {
		t.Fatalf("expected backend error from Marked, got %v", err)
	}

	err = manager.Clear(context.Background(), "event:4")
	if !errors.Is(err, backendErr) {
		t.Fatalf("expected backend error from Clear, got %v", err)
	}
}
