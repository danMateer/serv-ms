package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andres-erbsen/clock"
	"github.com/stretchr/testify/assert"
)

func TestUnsupportedMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handle(newMetrics())))
	defer server.Close()

	unsupportedMethods := []string{
		http.MethodPut,
		http.MethodPatch,
	}
	for _, method := range unsupportedMethods {
		req, _ := http.NewRequest(method, server.URL+"/metric/", nil)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

func TestLegalGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handle(newMetrics())))
	defer server.Close()

	tests := []struct {
		url            string
		contentType    string
		wantStatusCode int
	}{
		{
			url:            "/metric/foo/sum",
			contentType:    contentType,
			wantStatusCode: http.StatusOK,
		},
		{
			url:            "/metric/bar/sum",
			contentType:    contentType,
			wantStatusCode: http.StatusOK,
		},
		{
			url:            "/metric/baz/sum",
			contentType:    "text/html",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			url:            "/unknown",
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
		{
			url:            "/metric/foo)/sum",
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
		{
			url:            "/metric/foo",
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
		{
			url:            "/metric/foo/",
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, server.URL+tt.url, nil)
			req.Header.Set("content-type", tt.contentType)
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode)

			if resp.StatusCode == http.StatusOK {
				decoder := json.NewDecoder(resp.Body)
				jsonMap := make(map[string]int64)
				err = decoder.Decode(&jsonMap)
				assert.NoError(t, err)
				_, ok := jsonMap[value]
				assert.True(t, ok)
			}
		})
	}
}

func TestLegalPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(handle(newMetrics())))
	defer server.Close()

	validBody := "{\"value\": 100}"

	tests := []struct {
		url            string
		body           string
		contentType    string
		wantStatusCode int
	}{
		{
			url:            "/metric/foo",
			body:           validBody,
			contentType:    contentType,
			wantStatusCode: http.StatusOK,
		},
		{
			url:            "/metric/bar",
			body:           validBody,
			contentType:    contentType,
			wantStatusCode: http.StatusOK,
		},
		{
			url:            "/metric/novalue",
			body:           "{\"other_value\": 100}",
			contentType:    contentType,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			url:            "/metric/badjson",
			body:           "{'value': 100}",
			contentType:    contentType,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			url:            "/metric/baz",
			body:           validBody,
			contentType:    "text/html",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			url:            "/unknown",
			body:           validBody,
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
		{
			url:            "/metric/foo/sum",
			body:           validBody,
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
		{
			url:            "/metric/foo)",
			body:           validBody,
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
		{
			url:            "/metric/foo/",
			body:           validBody,
			contentType:    contentType,
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			req, _ := http.NewRequest(
				http.MethodPost,
				server.URL+tt.url,
				strings.NewReader(tt.body),
			)
			req.Header.Set("content-type", tt.contentType)
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode)

			if resp.StatusCode == http.StatusOK {
				decoder := json.NewDecoder(resp.Body)
				jsonMap := make(map[string]int64)
				err = decoder.Decode(&jsonMap)
				assert.NoError(t, err)
				assert.Equal(t, 0, len(jsonMap))
			}
		})
	}
}

func TestMetricsSingleKey(t *testing.T) {
	clk := clock.NewMock()
	m := newMetrics(withClock(clk))

	key := "active_visitors"

	m.record(key, 4)
	assert.Equal(t, int64(4), m.sum(key))
	clk.Add(90 * time.Minute)

	m.record(key, 3)
	assert.Equal(t, int64(3), m.sum(key))
	clk.Add(30 * time.Minute)

	m.record(key, 7)
	assert.Equal(t, int64(10), m.sum(key))
	clk.Add(40 * time.Second)

	m.record(key, 2)
	assert.Equal(t, int64(12), m.sum(key))
	clk.Add(5 * time.Second)
	assert.Equal(t, int64(12), m.sum(key))
}

func TestMetricsMultiKey(t *testing.T) {
	clk := clock.NewMock()
	m := newMetrics(withClock(clk))

	key1 := "foo"
	key2 := "bar"
	key3 := "baz"

	m.record(key1, 10)
	m.record(key2, 15)
	m.record(key3, 20)

	assert.Equal(t, int64(10), m.sum(key1))
	assert.Equal(t, int64(15), m.sum(key2))
	assert.Equal(t, int64(20), m.sum(key3))

	clk.Add(30 * time.Minute)

	m.record(key1, 100)
	m.record(key2, 200)
	m.record(key3, 300)

	assert.Equal(t, int64(110), m.sum(key1))
	assert.Equal(t, int64(215), m.sum(key2))
	assert.Equal(t, int64(320), m.sum(key3))

	clk.Add(35 * time.Minute)

	assert.Equal(t, int64(100), m.sum(key1))
	assert.Equal(t, int64(200), m.sum(key2))
	assert.Equal(t, int64(300), m.sum(key3))
}

func TestMetricsConcurrent(t *testing.T) {
	clk := clock.NewMock()
	m := newMetrics(withClock(clk))

	key := "active_visitors"

	numRoutines := 100
	for i := 0; i < numRoutines; i++ {
		go func() { m.record(key, 1) }()
	}
	gosched()

	assert.Equal(t, int64(numRoutines), m.sum(key))
}

func gosched() {
	time.Sleep(time.Millisecond)
}
