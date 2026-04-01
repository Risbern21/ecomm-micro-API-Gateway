package main

import (
	"context"
	"io"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerGracefulShutdown(t *testing.T) {
	server := &http.Server{
		Addr: ":54321",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.Write([]byte("completed"))
		}),
	}

	serverErrCh := make(chan error)
	go func() {
		serverErrCh <- runServer(context.Background(), server, 5*time.Second)
	}()

	resp, err := http.Get("http://localhost" + server.Addr)
	require.Equal(t, nil, err, "err should be nil")

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "test response status code")

	body, err := io.ReadAll(resp.Body)
	require.Equal(t, nil, err, "check error while reading response body")

	assert.Equal(t, "completed", string(body), "check if response is 'completed'")

	serverErr := <-serverErrCh
	require.Equal(t, nil, serverErr, "check if any server error")
}

func TestServerTimeoutShutdown(t *testing.T) {
	server := &http.Server{
		Addr: ":54322",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Second)
			w.Write([]byte("completed"))
		}),
	}

	serverErrCh := make(chan error)
	go func() {
		serverErrCh <- runServer(context.Background(), server, 5*time.Millisecond)
	}()

	requestErrCh := make(chan error)
	go func() {
		_, err := http.Get("http://localhost" + server.Addr)
		requestErrCh <- err
	}()

	time.Sleep(1 * time.Second)

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	assert.Errorf(t, <-requestErrCh, "check client request errors")

	serverErr := <-serverErrCh
	assert.Equal(t, serverErr, context.DeadlineExceeded, "check type of server error")
}
