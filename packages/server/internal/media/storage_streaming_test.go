package media

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestS3Provider_PutObject_StreamsWithUnsignedPayload is the core P3-#9a
// regression. Before the fix, PutObject did io.ReadAll on the caller's body
// and computed SHA256 of the bytes, peaking at 2× the object size in RAM.
// The fix switches to `x-amz-content-sha256: UNSIGNED-PAYLOAD`, which lets
// the HTTP transport stream the body directly. This test drives a real
// httptest server (no SigV4 verification, just header/body inspection) and
// asserts:
//
//  1. The signed request carries `x-amz-content-sha256: UNSIGNED-PAYLOAD`.
//  2. The body arrives at the server exactly as written (proving streaming
//     did not corrupt the payload).
//  3. No caller-visible copy of the bytes needed to be buffered — implicit,
//     since PutObject accepts any io.Reader including ones that cannot be
//     rewound.
func TestS3Provider_PutObject_StreamsWithUnsignedPayload(t *testing.T) {
	payload := []byte("cyimewrite-streaming-payload-marker")

	type capture struct {
		method      string
		contentHash string
		contentType string
		body        []byte
	}
	captured := make(chan capture, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured <- capture{
			method:      r.Method,
			contentHash: r.Header.Get("x-amz-content-sha256"),
			contentType: r.Header.Get("Content-Type"),
			body:        body,
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := &s3CompatibleProvider{
		name:        "s3",
		endpoint:    server.URL,
		bucket:      "test-bucket",
		region:      "auto",
		accessKeyID: "AKIA-TEST",
		secretKey:   "secret-test",
		httpClient:  server.Client(),
	}

	// Use a non-seekable reader to prove streaming really happens.
	body := &nonSeekableReader{Reader: bytes.NewReader(payload)}
	res, err := p.PutObject(context.Background(), PutObjectInput{
		ObjectKey:   "obj.png",
		ContentType: "image/png",
		Body:        body,
	})
	if err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if res == nil || res.Provider != "s3" {
		t.Fatalf("unexpected result: %+v", res)
	}

	got := <-captured
	if got.method != http.MethodPut {
		t.Fatalf("method = %q, want PUT", got.method)
	}
	if got.contentHash != unsignedPayloadHash {
		t.Fatalf("x-amz-content-sha256 = %q, want %q (streaming opt-in)", got.contentHash, unsignedPayloadHash)
	}
	if got.contentType != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got.contentType)
	}
	if !bytes.Equal(got.body, payload) {
		t.Fatalf("server received %q, want %q", got.body, payload)
	}
}

// TestS3Provider_PutObject_PropagatesFailureBody asserts that upstream errors
// still surface with a useful message, even though we LimitReader the body
// in the fix to prevent an unbounded error buffer.
func TestS3Provider_PutObject_PropagatesFailureBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("AccessDenied: bad key"))
	}))
	defer server.Close()

	p := &s3CompatibleProvider{
		name:        "s3",
		endpoint:    server.URL,
		bucket:      "b",
		region:      "auto",
		accessKeyID: "k",
		secretKey:   "s",
		httpClient:  server.Client(),
	}

	_, err := p.PutObject(context.Background(), PutObjectInput{
		ObjectKey:   "obj.png",
		ContentType: "image/png",
		Body:        bytes.NewReader([]byte("data")),
	})
	if err == nil {
		t.Fatal("expected error on 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected status code in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Fatalf("expected upstream body snippet in error, got %v", err)
	}
}

// TestS3Provider_GetObject_UsesEmptyPayloadHash asserts the GET/HEAD/DELETE
// path did *not* regress to UNSIGNED-PAYLOAD. Some minimal S3 implementations
// only accept the real zero-length SHA256 for body-less requests, so the fix
// must keep that branch distinct.
func TestS3Provider_GetObject_UsesEmptyPayloadHash(t *testing.T) {
	captured := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.Header.Get("x-amz-content-sha256")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	p := &s3CompatibleProvider{
		name:        "s3",
		endpoint:    server.URL,
		bucket:      "b",
		region:      "auto",
		accessKeyID: "k",
		secretKey:   "s",
		httpClient:  server.Client(),
	}

	obj, err := p.GetObject(context.Background(), "obj.png")
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	_, _ = io.Copy(io.Discard, obj.Body)
	_ = obj.Body.Close()

	hash := <-captured
	if hash != emptyPayloadSHA256 {
		t.Fatalf("x-amz-content-sha256 = %q, want %q", hash, emptyPayloadSHA256)
	}
}

// nonSeekableReader wraps an io.Reader and deliberately does NOT implement
// io.Seeker, so any code path that tries to rewind the body will fail to
// compile or panic — proving we genuinely stream instead of buffering and
// seeking.
type nonSeekableReader struct {
	io.Reader
}
