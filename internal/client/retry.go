package client

import (
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"time"
)

// DefaultRetries is how many times a client retries a transient failure. The
// --retries flag sets it before any client is created.
var DefaultRetries = 3

// retryBaseDelay is the first backoff delay; each retry doubles it.
var retryBaseDelay = 500 * time.Millisecond

// maxRetryDelay caps a single wait, including a server's Retry-After.
const maxRetryDelay = 30 * time.Second

// retryDelay returns how long to wait before retry number attempt (0-based),
// using the server's Retry-After seconds when given.
func retryDelay(attempt int, retryAfter string) time.Duration {
	if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
		return min(time.Duration(secs)*time.Second, maxRetryDelay)
	}
	backoff := retryBaseDelay << attempt
	jitter := time.Duration(rand.Int64N(int64(backoff)/2 + 1))
	return min(backoff+jitter, maxRetryDelay)
}

// isIdempotent reports whether repeating the request can't do the work twice.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return false
}

// shouldRetryStatus reports whether a response status is worth retrying.
// 429 and 503 mean the server turned the request away without doing it, so
// any request can be retried. Other gateway errors may arrive after the work
// was done (and charged), so only idempotent requests retry them.
func shouldRetryStatus(method string, status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return true
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return isIdempotent(method)
	}
	return false
}

// shouldRetryError reports whether a transport error is worth retrying. A
// failure to connect means the request never reached the server; a timeout
// may mean it ran, so only idempotent requests retry it.
func shouldRetryError(method string, err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return isIdempotent(method)
	}
	return false
}
