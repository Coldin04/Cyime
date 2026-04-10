package media

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalStorageProvider_SafePathRejectsEscapes covers the P2-#6 regression:
// even though current object keys are server-generated, the provider must
// refuse any key whose resolved path would land outside rootDir. Each case
// below is an attack primitive a future caller might accidentally enable.
func TestLocalStorageProvider_SafePathRejectsEscapes(t *testing.T) {
	rootDir := t.TempDir()
	p := &localStorageProvider{rootDir: rootDir, baseURL: "/media-files"}

	cases := []struct {
		name      string
		objectKey string
	}{
		{"empty key", ""},
		{"parent reference", ".."},
		{"parent with suffix", "../etc/passwd"},
		{"nested parent", "foo/../../etc/passwd"},
		{"null byte", "good\x00bad"},
		{"absolute unix", "/etc/passwd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := p.safePath(tc.objectKey); err == nil {
				t.Fatalf("expected safePath to reject %q", tc.objectKey)
			}
		})
	}
}

func TestLocalStorageProvider_SafePathAcceptsNormal(t *testing.T) {
	rootDir := t.TempDir()
	p := &localStorageProvider{rootDir: rootDir, baseURL: "/media-files"}

	got, err := p.safePath("user-id/20260101/asset.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rootAbs, _ := filepath.Abs(rootDir)
	want := filepath.Join(rootAbs, "user-id", "20260101", "asset.png")
	if got != want {
		t.Fatalf("safePath returned %q, want %q", got, want)
	}
}

// TestLocalStorageProvider_PutObjectRejectsEscape proves the guard is plumbed
// into the public PutObject entrypoint, not only the private helper.
func TestLocalStorageProvider_PutObjectRejectsEscape(t *testing.T) {
	rootDir := t.TempDir()
	p := &localStorageProvider{rootDir: rootDir, baseURL: "/media-files"}

	_, err := p.PutObject(context.TODO(), PutObjectInput{
		ObjectKey:   "../evil.png",
		ContentType: "image/png",
		Body:        bytes.NewReader([]byte{0x89, 0x50, 0x4e, 0x47}),
	})
	if err == nil {
		t.Fatal("expected PutObject to reject parent-traversal key")
	}
	if !errors.Is(err, errLocalStorageEscape) {
		t.Fatalf("expected errLocalStorageEscape, got %v", err)
	}

	// And crucially, nothing should have been written above rootDir.
	parent := filepath.Dir(rootDir)
	entries, _ := os.ReadDir(parent)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "evil") {
			t.Fatalf("escape wrote file %q in parent dir %q", entry.Name(), parent)
		}
	}
}

func TestLocalStorageProvider_GetAndDeleteRejectEscape(t *testing.T) {
	rootDir := t.TempDir()
	p := &localStorageProvider{rootDir: rootDir, baseURL: "/media-files"}

	if _, err := p.GetObject(context.TODO(), "../../etc/passwd"); !errors.Is(err, errLocalStorageEscape) {
		t.Fatalf("GetObject should reject escape, got %v", err)
	}
	if err := p.DeleteObject(context.TODO(), "../../etc/passwd"); !errors.Is(err, errLocalStorageEscape) {
		t.Fatalf("DeleteObject should reject escape, got %v", err)
	}
}

// TestLocalStorageProvider_PutGetDeleteRoundTrip is the happy-path regression:
// safe keys still work end-to-end after the defensive guard.
func TestLocalStorageProvider_PutGetDeleteRoundTrip(t *testing.T) {
	rootDir := t.TempDir()
	p := &localStorageProvider{rootDir: rootDir, baseURL: "/media-files"}

	_, err := p.PutObject(context.TODO(), PutObjectInput{
		ObjectKey:   "u1/day/obj.png",
		ContentType: "image/png",
		Body:        bytes.NewReader([]byte("data")),
	})
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	obj, err := p.GetObject(context.TODO(), "u1/day/obj.png")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer obj.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(obj.Body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	if buf.String() != "data" {
		t.Fatalf("unexpected body: %q", buf.String())
	}

	if err := p.DeleteObject(context.TODO(), "u1/day/obj.png"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
