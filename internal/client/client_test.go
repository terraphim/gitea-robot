// Copyright 2026 The Terraphim Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.terraphim.cloud/terraphim/gitea-robot/internal/config"
)

func newTestConfig() *config.Config {
	return &config.Config{
		BaseURL:          "http://example.com",
		Token:            "test-token",
		Timeout:          5 * time.Second,
		MaxResponseBytes: 1024 * 1024,
	}
}

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.BaseURL = srv.URL
	c := NewHTTPClient(cfg)

	data, err := c.Get(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"status":"ok"}` {
		t.Fatalf("expected ok, got %s", string(data))
	}
}

func TestGet_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.BaseURL = srv.URL
	c := NewHTTPClient(cfg)

	_, err := c.Get(context.Background(), srv.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestPost_Success(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.BaseURL = srv.URL
	c := NewHTTPClient(cfg)

	data, err := c.Post(context.Background(), srv.URL+"/test", []byte(`{"title":"test"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody != `{"title":"test"}` {
		t.Fatalf("expected body to be forwarded, got %s", receivedBody)
	}
	if string(data) != `{"id":1}` {
		t.Fatalf("expected response, got %s", string(data))
	}
}

func TestPatch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"state":"closed"}`))
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.BaseURL = srv.URL
	c := NewHTTPClient(cfg)

	data, err := c.Patch(context.Background(), srv.URL+"/test", []byte(`{"state":"closed"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"state":"closed"}` {
		t.Fatalf("unexpected response: %s", string(data))
	}
}

func TestDelete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.BaseURL = srv.URL
	c := NewHTTPClient(cfg)

	data, err := c.Delete(context.Background(), srv.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty body, got %s", string(data))
	}
}

func TestGet_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.Timeout = 50 * time.Millisecond
	cfg.BaseURL = srv.URL
	c := NewHTTPClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, srv.URL+"/test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
