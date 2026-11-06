package datacache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func tryGetData(t *testing.T, cache *DataCache[int], id int, sleep int, expected interface{}) {
	time.Sleep(time.Duration(sleep) * time.Millisecond)
	t.Logf("[%d] %v:start_get_data\n", id, time.Now().UnixNano()/int64(time.Millisecond))
	value := cache.Get()
	t.Logf("[%d] %v:got_data:%v\n", id, time.Now().UnixNano()/int64(time.Millisecond), value)
	if value != expected {
		t.Errorf("[%d] ERROR - get_data:%v!=%v", id, value, expected)
	}
}

func tryGetDataPtr(t *testing.T, cache *DataCache[*int], id int, sleep int, expected interface{}) {
	time.Sleep(time.Duration(sleep) * time.Millisecond)
	t.Logf("[%d] %v:start_get_data\n", id, time.Now().UnixNano()/int64(time.Millisecond))
	value := cache.Get()
	if value == nil {
		t.Logf("[%d] %v:got_data:%v\n", id, time.Now().UnixNano()/int64(time.Millisecond), value)
	} else {
		t.Logf("[%d] %v:got_data:%v\n", id, time.Now().UnixNano()/int64(time.Millisecond), *value)
	}
	if expected == nil {
		if value != nil {
			t.Errorf("[%d] ERROR - get_data:%v!=%v", id, value, expected)
		}
		return
	} else {
		if value == nil {
			t.Errorf("[%d] ERROR - get_data:%v!=%v", id, value, expected)
		} else if *value != expected {
			t.Errorf("[%d] ERROR - get_data:%v!=%v", id, *value, expected)
		}
	}
}

var errTestLoad = errors.New("test_load_error")

type errorWant int

const (
	wantNoError    errorWant = iota
	wantNotLoaded            // ErrNotLoaded without a load error
	wantLoadFailed           // ErrNotLoaded wrapping errTestLoad
)

func checkError(t *testing.T, err error, want errorWant) {
	t.Helper()
	switch want {
	case wantNoError:
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	case wantNotLoaded:
		if !errors.Is(err, ErrNotLoaded) || errors.Is(err, errTestLoad) {
			t.Errorf("err = %v, want ErrNotLoaded without the load error", err)
		}
	case wantLoadFailed:
		if !errors.Is(err, ErrNotLoaded) || !errors.Is(err, errTestLoad) {
			t.Errorf("err = %v, want ErrNotLoaded wrapping the load error", err)
		}
	}
}

func pollUntil(t *testing.T, what string, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatalf("%s did not happen within 10s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

type loadResult[V any] struct {
	value V
	err   error
}

// scriptedLoader holds each load until the test sends its result, and signals on started when a load begins
type scriptedLoader[V any] struct {
	started chan struct{}
	results chan loadResult[V]
}

func newScriptedLoader[V any](t *testing.T) *scriptedLoader[V] {
	l := &scriptedLoader[V]{
		started: make(chan struct{}, 100),
		results: make(chan loadResult[V]),
	}
	t.Cleanup(func() { close(l.results) })
	return l
}

func (l *scriptedLoader[V]) load() (V, error) {
	select {
	case l.started <- struct{}{}:
	default:
	}
	r := <-l.results
	return r.value, r.err
}

// waitStart calls get until a load starts
func (l *scriptedLoader[V]) waitStart(t *testing.T, get func()) {
	t.Helper()
	pollUntil(t, "load start", func() bool {
		select {
		case <-l.started:
			return true
		default:
			get()
			return false
		}
	})
}

// finish completes the running load with a result and waits until the next load starts
// With Expiration and ExpirationOnErr shorter than RetryInterval, the next load can start only after the result is stored
func (l *scriptedLoader[V]) finish(t *testing.T, value V, err error, get func()) {
	t.Helper()
	l.results <- loadResult[V]{value: value, err: err}
	l.waitStart(t, get)
}

func countLoads[T any](load LoadDataCallback[T], loads *atomic.Int64) LoadDataCallback[T] {
	return func() (T, error) {
		loads.Add(1)
		return load()
	}
}

func TestDataCache(t *testing.T) {
	// with init data
	l := newScriptedLoader[int](t)
	cache := NewDataCache(&CacheInitOption[int]{
		InitData:   0,
		LoadData:   l.load,
		Expiration: 50,
	})
	get := func() { cache.Get() }
	tryGetData(t, cache, 1, 0, 0)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1}
	pollUntil(t, "load", func() bool { return cache.Get() != 0 })
	tryGetData(t, cache, 2, 0, 1)
	tryGetData(t, cache, 3, 70, 1)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 2}
	pollUntil(t, "reload", func() bool { return cache.Get() != 1 })
	tryGetData(t, cache, 4, 0, 2)

	// without init data
	l = newScriptedLoader[int](t)
	cache = NewDataCache(&CacheInitOption[int]{
		LoadData:    l.load,
		Expiration:  20,
		WaitTimeout: 1,
	})
	tryGetData(t, cache, 5, 0, 0)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1}
	pollUntil(t, "load", func() bool { return cache.Get() != 0 })
	tryGetData(t, cache, 6, 0, 1)

	// without init data (loadData* func return err without cache on err)
	var loads atomic.Int64
	l = newScriptedLoader[int](t)
	cache = NewDataCache(&CacheInitOption[int]{
		LoadData:    countLoads(l.load, &loads),
		Expiration:  20,
		WaitTimeout: 1,
	})
	tryGetData(t, cache, 7, 0, 0)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", func() bool {
		_, err := cache.GetWithError()
		return errors.Is(err, errTestLoad)
	})
	tryGetData(t, cache, 8, 0, 0)
	if n := loads.Load(); n != 1 {
		t.Errorf("loads should be 1 (no retry within RetryInterval), but is %v", n)
	}

	// without init data (loadData* func return err with cache on err)
	l = newScriptedLoader[int](t)
	cache = NewDataCache(&CacheInitOption[int]{
		InitData:        0,
		LoadData:        l.load,
		Expiration:      50,
		ExpirationOnErr: 40,
	})
	tryGetData(t, cache, 9, 0, 0)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", func() bool { return cache.Get() != 0 })
	tryGetData(t, cache, 10, 0, 1)
	tryGetData(t, cache, 11, 70, 1)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 2, err: errTestLoad}
	pollUntil(t, "failed reload", func() bool { return cache.Get() != 1 })
	tryGetData(t, cache, 12, 0, 2)

	// without init data (loadData* func return err with cache on err)
	l = newScriptedLoader[int](t)
	cache = NewDataCache(&CacheInitOption[int]{
		LoadData:        l.load,
		Expiration:      20,
		ExpirationOnErr: 10,
		WaitTimeout:     1,
	})
	tryGetData(t, cache, 13, 0, 0)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", func() bool { return cache.Get() != 0 })
	tryGetData(t, cache, 14, 0, 1)
}

func TestDataNilCache(t *testing.T) {
	// without init data (loadData* func return err and nil as to cache data with cache on err)
	var loads atomic.Int64
	l := newScriptedLoader[*int](t)
	cache := NewDataCache(&CacheInitOption[*int]{
		LoadData:        countLoads(l.load, &loads),
		Expiration:      20,
		ExpirationOnErr: 60000,
		WaitTimeout:     1,
	})
	tryGetDataPtr(t, cache, 15, 0, nil)
	l.waitStart(t, func() { cache.Get() })
	l.results <- loadResult[*int]{err: errTestLoad}
	pollUntil(t, "failed load", func() bool {
		_, err := cache.GetWithError()
		return errors.Is(err, errTestLoad)
	})
	tryGetDataPtr(t, cache, 16, 0, nil)
	if n := loads.Load(); n != 1 {
		t.Errorf("loads should be 1, but is %v", n)
	}
}

func TestDataPtrCache(t *testing.T) {
	// with init data
	l := newScriptedLoader[*int](t)
	cache := NewDataCache(&CacheInitOption[*int]{
		InitData:   nil,
		LoadData:   l.load,
		Expiration: 50,
	})
	get := func() { cache.Get() }
	first, second := 1, 2
	tryGetDataPtr(t, cache, 1, 0, nil)
	l.waitStart(t, get)
	l.results <- loadResult[*int]{value: &first}
	pollUntil(t, "load", func() bool { return cache.Get() != nil })
	tryGetDataPtr(t, cache, 2, 0, 1)
	tryGetDataPtr(t, cache, 3, 70, 1)
	l.waitStart(t, get)
	l.results <- loadResult[*int]{value: &second}
	pollUntil(t, "reload", func() bool { return cache.Get() != &first })
	tryGetDataPtr(t, cache, 4, 0, 2)
}

func TestDataCacheGetConcurrent(t *testing.T) {
	var loads atomic.Int64
	cache := NewDataCache(&CacheInitOption[int]{
		LoadData: func() (int, error) {
			return int(loads.Add(1)), nil
		},
		Expiration:    1,
		RetryInterval: 1,
	})
	deadline := time.Now().Add(200 * time.Millisecond)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				if value := cache.Get(); value < 0 || int64(value) > loads.Load() {
					t.Errorf("Get = %v, want 0 or a value a load returned", value)
				}
			}
		}()
	}
	wg.Wait()
}

func TestDataCacheGetWithError(t *testing.T) {
	cases := []struct {
		name            string
		withInitData    bool
		initData        int
		value           int
		err             error
		block           bool
		expirationOnErr int64
		waitTimeout     int64
		waitsFull       bool
		wantValue       int
		wantErr         errorWant
		wantGet         int
	}{
		{name: "found", value: 7, waitTimeout: 10000, wantValue: 7, wantErr: wantNoError, wantGet: 7},
		{name: "not found", value: 0, waitTimeout: 10000, wantErr: wantNoError},
		{name: "first load failed, retry pending", value: 9, err: errTestLoad, waitTimeout: 300, waitsFull: true, wantErr: wantLoadFailed},
		{name: "first load failed with ExpirationOnErr", value: 9, err: errTestLoad, expirationOnErr: 60000, waitTimeout: 10000, wantErr: wantLoadFailed, wantGet: 9},
		{name: "first load not finished within WaitTimeout", block: true, waitTimeout: 20, waitsFull: true, wantErr: wantNotLoaded},
		{name: "WaitTimeout 0", block: true, wantErr: wantNotLoaded},
		{name: "WithInitData", withInitData: true, block: true, wantErr: wantNoError},
		{name: "non-zero InitData", initData: 5, block: true, wantValue: 5, wantErr: wantNoError, wantGet: 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			cache := NewDataCache(&CacheInitOption[int]{
				WithInitData: tc.withInitData,
				InitData:     tc.initData,
				LoadData: func() (int, error) {
					if tc.block {
						<-release
					}
					return tc.value, tc.err
				},
				Expiration:      60000,
				ExpirationOnErr: tc.expirationOnErr,
				RetryInterval:   60000,
				WaitTimeout:     tc.waitTimeout,
			})
			start := time.Now()
			value, err := cache.GetWithError()
			if elapsed := time.Since(start); tc.waitsFull && elapsed < time.Duration(tc.waitTimeout)*time.Millisecond {
				t.Errorf("GetWithError returned after %v, before WaitTimeout %dms", elapsed, tc.waitTimeout)
			}
			if value != tc.wantValue {
				t.Errorf("value = %v, want %v", value, tc.wantValue)
			}
			checkError(t, err, tc.wantErr)
			if got := cache.Get(); got != tc.wantGet {
				t.Errorf("Get = %v, want %v", got, tc.wantGet)
			}
		})
	}
}

func TestDataCacheGetWithErrorRetryRunning(t *testing.T) {
	l := newScriptedLoader[int](t)
	cache := NewDataCache(&CacheInitOption[int]{
		LoadData:      l.load,
		Expiration:    60000,
		RetryInterval: 100,
	})
	get := func() { cache.GetWithError() }
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 9, err: errTestLoad}
	pollUntil(t, "first load failure", func() bool {
		_, err := cache.GetWithError()
		return errors.Is(err, errTestLoad)
	})

	l.waitStart(t, get)
	value, err := cache.GetWithError()
	if value != 0 {
		t.Errorf("value while retry running = %v, want 0", value)
	}
	checkError(t, err, wantLoadFailed)

	l.results <- loadResult[int]{value: 7}
	pollUntil(t, "retry success", func() bool {
		value, err := cache.GetWithError()
		return value == 7 && err == nil
	})
}

func TestDataCacheGetWithErrorRefreshFailed(t *testing.T) {
	cases := []struct {
		name      string
		initData  int
		firstLoad int
		want      int
	}{
		{name: "after a successful load", firstLoad: 7, want: 7},
		{name: "with init data", initData: 5, want: 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := newScriptedLoader[int](t)
			cache := NewDataCache(&CacheInitOption[int]{
				InitData:      tc.initData,
				LoadData:      l.load,
				Expiration:    1,
				RetryInterval: 60000,
			})
			get := func() { cache.GetWithError() }
			l.waitStart(t, get)
			if tc.firstLoad != 0 {
				l.finish(t, tc.firstLoad, nil, get)
			}

			l.results <- loadResult[int]{value: 9, err: errTestLoad}
			for i := 0; i < 50; i++ {
				value, err := cache.GetWithError()
				if value != tc.want || err != nil {
					t.Fatalf("GetWithError after failed load = (%v, %v), want (%v, nil)", value, err, tc.want)
				}
				time.Sleep(time.Millisecond)
			}
			if got := cache.Get(); got != tc.want {
				t.Errorf("Get = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDataCacheGetWithErrorExpirationOnErr(t *testing.T) {
	l := newScriptedLoader[int](t)
	cache := NewDataCache(&CacheInitOption[int]{
		LoadData:        l.load,
		Expiration:      1,
		ExpirationOnErr: 1,
		RetryInterval:   60000,
	})
	get := func() { cache.GetWithError() }
	l.waitStart(t, get)
	l.finish(t, 7, nil, get)
	value, err := cache.GetWithError()
	if value != 7 {
		t.Errorf("value after success = %v, want 7", value)
	}
	checkError(t, err, wantNoError)

	l.finish(t, 9, errTestLoad, get)
	value, err = cache.GetWithError()
	if value != 0 {
		t.Errorf("value after failure = %v, want 0", value)
	}
	checkError(t, err, wantLoadFailed)
	if got := cache.Get(); got != 9 {
		t.Errorf("Get after failure = %v, want 9", got)
	}

	l.finish(t, 8, nil, get)
	value, err = cache.GetWithError()
	if value != 8 {
		t.Errorf("value after recovery = %v, want 8", value)
	}
	checkError(t, err, wantNoError)
}

func TestDataCacheGetWithErrorConcurrent(t *testing.T) {
	cases := []struct {
		name            string
		expirationOnErr int64
	}{
		{name: "without ExpirationOnErr"},
		{name: "with ExpirationOnErr", expirationOnErr: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var loads atomic.Int64
			cache := NewDataCache(&CacheInitOption[int]{
				LoadData: func() (int, error) {
					n := loads.Add(1)
					if n%2 == 0 {
						return -1, errTestLoad
					}
					return int(n), nil
				},
				Expiration:      1,
				ExpirationOnErr: tc.expirationOnErr,
				RetryInterval:   1,
			})
			deadline := time.Now().Add(200 * time.Millisecond)
			var wg sync.WaitGroup
			for g := 0; g < 8; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for time.Now().Before(deadline) {
						value, err := cache.GetWithError()
						if err != nil {
							if value != 0 || !errors.Is(err, ErrNotLoaded) {
								t.Errorf("GetWithError = (%v, %v), want zero value and ErrNotLoaded", value, err)
							}
						} else if value <= 0 {
							t.Errorf("GetWithError = (%v, nil), want a successful load", value)
						}
					}
				}()
			}
			wg.Wait()
		})
	}
}
