package client

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func fastRetries(t *testing.T) {
	t.Helper()
	old := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = old })
}

// flakyServer fails the first n requests with status, then succeeds.
func flakyServer(t *testing.T, n int32, status int, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= n {
			w.WriteHeader(status)
			return
		}
		writeJSONResponse(w, http.StatusOK, []any{})
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestRetriesTooManyRequestsThenSucceeds(t *testing.T) {
	fastRetries(t)
	var calls atomic.Int32
	ts := flakyServer(t, 2, http.StatusTooManyRequests, &calls)

	resp, err := New(ts.URL, "k").post("/api/v1/pipelines/x/evaluate", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || calls.Load() != 3 {
		t.Fatalf("status %d after %d calls, want 200 after 3", resp.StatusCode, calls.Load())
	}
}

func TestRetriesServiceUnavailableForPost(t *testing.T) {
	fastRetries(t)
	var calls atomic.Int32
	ts := flakyServer(t, 1, http.StatusServiceUnavailable, &calls)

	resp, err := New(ts.URL, "k").post("/x", []byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if calls.Load() != 2 {
		t.Fatalf("got %d calls, want 2", calls.Load())
	}
}

func TestDoesNotRetryBadGatewayForPost(t *testing.T) {
	// A 502 on a POST may mean the work ran (and was charged), so no retry.
	fastRetries(t)
	var calls atomic.Int32
	ts := flakyServer(t, 1, http.StatusBadGateway, &calls)

	resp, err := New(ts.URL, "k").post("/x", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway || calls.Load() != 1 {
		t.Fatalf("status %d after %d calls, want 502 after 1", resp.StatusCode, calls.Load())
	}
}

func TestRetriesBadGatewayForGet(t *testing.T) {
	fastRetries(t)
	var calls atomic.Int32
	ts := flakyServer(t, 1, http.StatusBadGateway, &calls)

	resp, err := New(ts.URL, "k").get("/x")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	fastRetries(t)
	var calls atomic.Int32
	ts := flakyServer(t, 1, http.StatusUnprocessableEntity, &calls)

	resp, err := New(ts.URL, "k").get("/x")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if calls.Load() != 1 {
		t.Fatalf("got %d calls, want 1", calls.Load())
	}
}

func TestGivesUpAfterRetries(t *testing.T) {
	fastRetries(t)
	var calls atomic.Int32
	ts := flakyServer(t, 100, http.StatusTooManyRequests, &calls)

	c := New(ts.URL, "k")
	c.Retries = 2
	resp, err := c.get("/x")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests || calls.Load() != 3 {
		t.Fatalf("status %d after %d calls, want 429 after 3", resp.StatusCode, calls.Load())
	}
}

func TestRetriesTimeoutForGetButNotPost(t *testing.T) {
	fastRetries(t)
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			time.Sleep(200 * time.Millisecond)
		}
		writeJSONResponse(w, http.StatusOK, nil)
	}))
	t.Cleanup(ts.Close)

	c := New(ts.URL, "k")
	c.httpClient.Timeout = 50 * time.Millisecond
	resp, err := c.get("/x")
	if err != nil {
		t.Fatalf("GET should have been retried after the timeout: %v", err)
	}
	resp.Body.Close()

	calls.Store(0)
	if _, err := c.post("/x", []byte(`{}`)); err == nil {
		t.Fatal("POST timeout should not be retried")
	}
	if calls.Load() != 1 {
		t.Fatalf("POST made %d calls, want 1", calls.Load())
	}
}

func TestRetryAfterHeaderIsHonoured(t *testing.T) {
	if d := retryDelay(0, "2"); d != 2*time.Second {
		t.Fatalf("Retry-After: 2 gave %v", d)
	}
	if d := retryDelay(0, "600"); d != maxRetryDelay {
		t.Fatalf("a long Retry-After should be capped at %v, got %v", maxRetryDelay, d)
	}
}
