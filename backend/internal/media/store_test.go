package media

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestMemoryStoreRejectsInvalidTypeAndTraversal(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if err := store.Put(ctx, "../x", "image/png", bytes.NewReader([]byte("x")), 1); err != ErrInvalidKey {
		t.Fatalf("key: %v", err)
	}
	if err := store.Put(ctx, "ok", "application/x-msdownload", bytes.NewReader([]byte("x")), 1); err != ErrInvalidType {
		t.Fatalf("type: %v", err)
	}
	body := []byte("hello-image")
	if err := store.Put(ctx, "att1.png", "image/png", bytes.NewReader(body), int64(len(body))); err != nil {
		t.Fatal(err)
	}
	r, size, err := store.Open(ctx, "att1.png")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(r)
	_ = r.Close()
	if size != int64(len(body)) || string(got) != "hello-image" {
		t.Fatalf("got %q size %d", got, size)
	}
}
