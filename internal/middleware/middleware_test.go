package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/risbern21/api_gateway/internal/cache"
	"github.com/risbern21/api_gateway/internal/logger"
	"github.com/risbern21/api_gateway/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	cache.Connect()
	logger.InitLogger()

	exitVal := m.Run()
	os.Exit(exitVal)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestMiddleware(t *testing.T) *Middleware {
	t.Helper()
	return NewMiddleware("test-secret-key-that-is-long-enough!!")
}

func validToken(t *testing.T) string {
	t.Helper()
	maker := token.NewJWTMaker("test-secret-key-that-is-long-enough!!")
	tok, _, err := maker.CreateToken(uuid.New(), "test@example.com", "user", time.Minute)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return tok
}

// flushKeys deletes the given Redis keys after the test completes so
// individual tests do not bleed into each other.
func flushKeys(t *testing.T, keys ...string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		for _, k := range keys {
			cache.Client().Del(ctx, k)
		}
	})
}

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
})

// ─── getIP ───────────────────────────────────────────────────────────────────

func TestGetIP_FromXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.1")

	ip, err := getIP(r)
	require.NoError(t, err, "must return no error")

	fmt.Println("got ip ", ip)

	assert.Equal(t, "198.51.100.1", ip, "ip's must be equal")
}

func TestGetIP_FromRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.42:1234"

	ip, err := getIP(r)
	require.NoError(t, err, "must return no error")

	assert.Equal(t, "192.0.2.42", ip, "ip's must be equal")
}

func TestGetIP_Loopback(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "[::1]:9090"

	ip, err := getIP(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "127.0.0.1" {
		t.Errorf("got %q, want 127.0.0.1", ip)
	}
}

func TestGetIP_InvalidRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "not-an-address"

	_, err := getIP(r)
	if err == nil {
		t.Fatal("expected error for invalid RemoteAddr, got nil")
	}
}

// ─── ResponseRecorder ────────────────────────────────────────────────────────

func TestResponseRecorder_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &ResponseRecorder{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}

	n, err := rec.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("Write returned %d, want 5", n)
	}
	if rec.body.String() != "hello" {
		t.Errorf("body buffer = %q, want \"hello\"", rec.body.String())
	}
}

func TestResponseRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &ResponseRecorder{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}

	rec.WriteHeader(http.StatusCreated)

	if rec.statusCode != http.StatusCreated {
		t.Errorf("statusCode = %d, want %d", rec.statusCode, http.StatusCreated)
	}
	if w.Code != http.StatusCreated {
		t.Errorf("underlying recorder code = %d, want %d", w.Code, http.StatusCreated)
	}
}

// ─── RateLimitingMiddleware ───────────────────────────────────────────────────

func TestRateLimitingMiddleware_AllowsUnderLimit(t *testing.T) {
	ip := "10.1.0.1"
	flushKeys(t, ip)

	m := newTestMiddleware(t)
	handler := m.RateLimitingMiddleware(okHandler)

	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":1234"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want %d", i, rr.Code, http.StatusOK)
		}
	}
}

func TestRateLimitingMiddleware_BlocksOverLimit(t *testing.T) {
	ip := "10.1.0.2"
	flushKeys(t, ip)

	// Pre-seed the counter to 5 (the allowed limit).
	ctx := context.Background()
	cache.Client().Set(ctx, ip, "5", 15*time.Second)

	m := newTestMiddleware(t)
	handler := m.RateLimitingMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip + ":5678"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("got %d, want %d (TooManyRequests)", rr.Code, http.StatusTooManyRequests)
	}
}

// TestRateLimitingMiddleware_CounterExpiresAndResets waits for the real
// 15 s Redis TTL. It is marked with t.Skip if you want to exclude slow tests;
// remove the skip when running a full integration suite.
func TestRateLimitingMiddleware_CounterExpiresAndResets(t *testing.T) {
	ip := "10.1.0.3"
	flushKeys(t, ip)

	m := newTestMiddleware(t)
	handler := m.RateLimitingMiddleware(okHandler)

	makeReq := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":9999"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr.Code
	}

	// Exhaust the 5-request limit.
	for range 5 {
		makeReq()
	}
	if got := makeReq(); got != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit exhausted, got %d", got)
	}

	// Wait for the real 15 s TTL to expire.
	time.Sleep(16 * time.Second)

	if got := makeReq(); got != http.StatusOK {
		t.Errorf("after TTL expiry: got %d, want 200", got)
	}
}

// ─── AuthenticationMiddleware ────────────────────────────────────────────────

func TestAuthenticationMiddleware_NoHeader(t *testing.T) {
	m := newTestMiddleware(t)
	handler := m.AuthenticationMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddleware_MissingBearerPrefix(t *testing.T) {
	m := newTestMiddleware(t)
	handler := m.AuthenticationMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddleware_InvalidToken(t *testing.T) {
	m := newTestMiddleware(t)
	handler := m.AuthenticationMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer this.is.not.valid")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddleware_ExpiredToken(t *testing.T) {
	maker := token.NewJWTMaker("test-secret-key-that-is-long-enough!!")
	tok, _, _ := maker.CreateToken(uuid.New(), "expired@example.com", "user", -time.Minute) // already expired

	m := newTestMiddleware(t)
	handler := m.AuthenticationMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticationMiddleware_ValidToken(t *testing.T) {
	tok := validToken(t)
	m := newTestMiddleware(t)
	handler := m.AuthenticationMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rr.Code, http.StatusOK)
	}
}

// ─── CachingMiddleware ────────────────────────────────────────────────────────

func TestCachingMiddleware_CacheMiss_ThenPopulatesCache(t *testing.T) {
	key := constructKey(http.MethodGet, "/api/items")
	flushKeys(t, key)

	m := newTestMiddleware(t)
	callCount := 0
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"fresh"}`))
	})
	handler := m.CachingMiddleware(upstream)

	// First request – cache miss.
	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first request: got %d, want 200", rr.Code)
	}
	if callCount != 1 {
		t.Errorf("upstream called %d times on cache miss, want 1", callCount)
	}

	// Second request – must be served from cache (upstream not called again).
	req2 := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("second request: got %d, want 200", rr2.Code)
	}
	if callCount != 1 {
		t.Errorf("upstream called %d times – expected cache hit on 2nd request", callCount)
	}
}

func TestCachingMiddleware_DoesNotCacheNon200(t *testing.T) {
	key := constructKey(http.MethodGet, "/api/broken")
	flushKeys(t, key)

	m := newTestMiddleware(t)
	callCount := 0
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})
	handler := m.CachingMiddleware(upstream)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/broken", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	if callCount != 3 {
		t.Errorf("expected upstream called 3 times (non-200 not cached), got %d", callCount)
	}
}

// TestCachingMiddleware_CacheExpires waits for the real 60 s Redis TTL.
func TestCachingMiddleware_CacheExpires(t *testing.T) {
	key := constructKey(http.MethodGet, "/api/expire")
	flushKeys(t, key)

	m := newTestMiddleware(t)
	callCount := 0
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"call":` + string(rune('0'+callCount)) + `}`))
	})
	handler := m.CachingMiddleware(upstream)

	makeReq := func() {
		req := httptest.NewRequest(http.MethodGet, "/api/expire", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	makeReq() // populates cache with 60 s TTL

	// Wait for the real 60 s TTL to expire.
	time.Sleep(61 * time.Second)

	makeReq() // cache expired – upstream called again

	if callCount != 2 {
		t.Errorf("expected 2 upstream calls after TTL expiry, got %d", callCount)
	}
}

func TestCachingMiddleware_KeyIsolatedByMethodAndPath(t *testing.T) {
	keys := []string{
		constructKey(http.MethodGet, "/a"),
		constructKey(http.MethodPost, "/a"),
		constructKey(http.MethodGet, "/b"),
	}
	flushKeys(t, keys...)

	m := newTestMiddleware(t)
	callCount := 0
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("body"))
	})
	handler := m.CachingMiddleware(upstream)

	requests := []struct{ method, path string }{
		{http.MethodGet, "/a"},
		{http.MethodPost, "/a"},
		{http.MethodGet, "/b"},
	}
	for _, p := range requests {
		req := httptest.NewRequest(p.method, p.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	if callCount != 3 {
		t.Errorf("expected 3 upstream calls for 3 distinct cache keys, got %d", callCount)
	}
}

// ─── LoggingMiddleware ────────────────────────────────────────────────────────

func TestLoggingMiddleware_PassesThrough(t *testing.T) {
	m := newTestMiddleware(t)
	handler := m.LoggingMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "must return http.StatusOK")
	assert.Equal(t, "OK", rr.Body.String(), "must return \"OK\"")
}

func TestLoggingMiddleware_DoesNotAlterResponse(t *testing.T) {
	m := newTestMiddleware(t)
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})
	handler := m.LoggingMiddleware(customHandler)

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201", rr.Code)
	}
	if rr.Header().Get("X-Custom") != "yes" {
		t.Errorf("X-Custom header missing or wrong: %q", rr.Header().Get("X-Custom"))
	}
	if rr.Body.String() != "created" {
		t.Errorf("body = %q, want \"created\"", rr.Body.String())
	}
}

// ─── constructKey ────────────────────────────────────────────────────────────

func TestConstructKey(t *testing.T) {
	tests := []struct {
		method, path, want string
	}{
		{"GET", "/users", "GET-/users"},
		{"POST", "/orders/42", "POST-/orders/42"},
		{"DELETE", "/", "DELETE-/"},
	}
	for _, tc := range tests {
		got := constructKey(tc.method, tc.path)
		if got != tc.want {
			t.Errorf("constructKey(%q,%q) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}
