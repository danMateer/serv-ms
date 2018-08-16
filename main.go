package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/andres-erbsen/clock"
)

const contentType = "application/json"
const value = "value"

func main() {
	m := newMetrics()
	http.HandleFunc("/metric/", handle(m))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handle(m *metrics) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGet(m, w, r)
		case http.MethodPost:
			handlePost(m, w, r)
		default:
			w.Header().Set("Allow", fmt.Sprintf("%s, %s", http.MethodGet, http.MethodPost))
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleGet(m *metrics, w http.ResponseWriter, r *http.Request) {
	// Enforce content-type
	if r.Header.Get("content-type") != contentType {
		http.Error(w, fmt.Sprintf("Error: content-type MUST be %s", contentType), http.StatusBadRequest)
		return
	}

	// Enforce url schema
	rx, _ := regexp.Compile(`^/metric/(\w+)/sum$`)
	if !rx.MatchString(r.URL.Path) {
		http.Error(w, "Error: legal GET urls look like '/metric/{key}/sum'", http.StatusNotFound)
		return
	}

	// Get key
	match := rx.FindStringSubmatch(r.URL.Path)
	if match == nil || len(match) < 2 {
		return
	}
	key := match[1]

	// Return sum
	val := m.sum(key)
	encoder := json.NewEncoder(w)
	encoder.Encode(map[string]int64{
		value: val,
	})
}

func handlePost(m *metrics, w http.ResponseWriter, r *http.Request) {
	// Enforce content-type
	if r.Header.Get("content-type") != contentType {
		http.Error(w, fmt.Sprintf("Error: content-type MUST be %s", contentType), http.StatusBadRequest)
		return
	}

	// Enforce url schema
	rx, _ := regexp.Compile(`^/metric/(\w+)$`)
	if !rx.MatchString(r.URL.Path) {
		http.Error(w, "Error: legal POST urls look like '/metric/{key}'", http.StatusNotFound)
		return
	}

	// Get key
	match := rx.FindStringSubmatch(r.URL.Path)
	if match == nil || len(match) < 2 {
		return
	}
	key := match[1]

	// Get value
	decoder := json.NewDecoder(r.Body)
	jsonMap := make(map[string]int64)
	err := decoder.Decode(&jsonMap)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: invalid JSON: %s'", err), http.StatusBadRequest)
		return
	}
	val, ok := jsonMap[value]
	if !ok {
		http.Error(w, "Error: JSON must contain key named 'value': %s'", http.StatusBadRequest)
		return
	}

	// Record value
	m.record(key, val)
	fmt.Fprintln(w, "{}")
}

// We'll store incoming values in per-minute maps.
// Each per-minute "bucket" is a map of keys to running counts.
// Those running counts are the values seen so far for a given minute.
// In practice the count should only be "running" for the current minute.
type metrics struct {
	sync.Mutex // guard minutes

	clk     clock.Clock
	minutes map[int64]map[string]int64
}

// withClock and the clock package help us mock time for testing.
func withClock(clk clock.Clock) func(*metrics) {
	return func(m *metrics) {
		m.clk = clk
	}
}

// newMetrics returns a new metrics struct.
// This helps keep data independent between tests.
func newMetrics(options ...func(*metrics)) *metrics {
	m := &metrics{}
	m.minutes = make(map[int64]map[string]int64)

	for _, option := range options {
		option(m)
	}

	if m.clk == nil {
		m.clk = clock.New()
	}

	return m
}

// sum recorded values for a given key across the last hour.
func (m *metrics) sum(key string) int64 {
	m.Lock()
	defer m.Unlock()

	nowBucket := minutes(m.clk.Now())

	sum := int64(0)
	for bucketOffset := 0; bucketOffset < 60; bucketOffset++ {
		bucket := m.minutes[nowBucket-int64(bucketOffset)]
		if bucket != nil {
			sum += bucket[key]
		}
	}

	return sum
}

// record a key/value.
func (m *metrics) record(key string, value int64) int64 {
	m.Lock()
	defer m.Unlock()

	bucket := minutes(m.clk.Now())

	if m.minutes[bucket] == nil {
		m.minutes[bucket] = make(map[string]int64)
	}

	m.minutes[bucket][key] += value

	return m.minutes[bucket][key]
}

// minutes returns a minute bucket.
// We'll sum across 60 minute buckets to implement a sliding hour window.
func minutes(t time.Time) int64 {
	return t.Unix() / 60
}
