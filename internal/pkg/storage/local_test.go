package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocalProviderRejectsPathTraversal(t *testing.T) {
	p, err := NewLocalProvider(t.TempDir(), "/storage")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := p.Upload(context.Background(), "../escape.txt", strings.NewReader("x"), "text/plain"); err == nil {
		t.Fatal("expected traversal upload to fail")
	}
	if _, err := p.Download(context.Background(), "../escape.txt"); err == nil {
		t.Fatal("expected traversal download to fail")
	}
	if err := p.Delete(context.Background(), "../escape.txt"); err == nil {
		t.Fatal("expected traversal delete to fail")
	}
}
