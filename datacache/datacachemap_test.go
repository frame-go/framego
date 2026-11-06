package datacache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testData struct {
	K string
	V int
}

func scriptedLoadData(l *scriptedLoader[int]) LoadDataByKeyCallback[string, *testData] {
	return func(key string) (*testData, error) {
		v, err := l.load()
		return &testData{K: key, V: v}, err
	}
}

func scriptedLoadDataBatch(l *scriptedLoader[int]) LoadDataBatchByKeysCallback[string, *testData] {
	return func(keys []string) (map[string]*testData, error) {
		v, err := l.load()
		data := make(map[string]*testData)
		for _, key := range keys {
			data[key] = &testData{K: key, V: v}
		}
		return data, err
	}
}

func getManyReturns(cache *CacheMap[string, *testData], keys []string, v int) func() bool {
	return func() bool {
		for _, value := range cache.GetMany(keys) {
			if value == nil || value.V != v {
				return false
			}
		}
		return true
	}
}

func sleepGetBatchNilDataMapErr(keys []string) (map[string]*testData, error) {
	time.Sleep(10 * time.Millisecond)
	return nil, errors.New("dummy")
}

func sleepGetDataSimple(key string) (*testData, error) {
	time.Sleep(5 * time.Millisecond)
	return &testData{K: key, V: 0}, nil
}

func sleepGetManyDataSimple(keys []string) (map[string]*testData, error) {
	time.Sleep(5 * time.Millisecond)
	data := make(map[string]*testData)
	for _, key := range keys {
		data[key] = &testData{K: key, V: 0}
	}
	return data, nil
}

func tryGetDataMap(t *testing.T, cache *CacheMap[string, *testData], id int, sleep int, key string, expected interface{}) {
	time.Sleep(time.Duration(sleep) * time.Millisecond)
	t.Logf("[%d] %v:start_get_data(%s)\n", id, time.Now().UnixNano()/int64(time.Millisecond), key)
	value := cache.Get(key)
	if value == nil {
		t.Logf("[%d] %v:got_data(%s):%v\n", id, time.Now().UnixNano()/int64(time.Millisecond), key, value)
	} else {
		t.Logf("[%d] %v:got_data(%s):%v\n", id, time.Now().UnixNano()/int64(time.Millisecond), key, value.V)

	}
	if expected == nil {
		if value != nil {
			t.Errorf("[%d] ERROR - get_data(%s): expected nil, but %d", id, key, value.V)
		}
	} else {
		if value == nil {
			t.Errorf("[%d] ERROR - get_data(%s): expected %d, but nil", id, key, expected)
		} else {
			if value.V != expected {
				t.Errorf("[%d] ERROR - get_data(%s):%v!=%v", id, key, value.V, expected)
			}
		}
	}
}

func tryGetManyDataMap(t *testing.T, cache *CacheMap[string, *testData], id int, sleep int, keys []string, expected []interface{}) {
	time.Sleep(time.Duration(sleep) * time.Millisecond)
	data := cache.GetMany(keys)
	for idx, key := range keys {
		value := data[key]
		if expected[idx] == nil {
			if value != nil {
				t.Errorf("[%d] ERROR - get_data(%s): expected nil, but %d", id, key, value.V)
			}
		} else {
			if value == nil {
				t.Errorf("[%d] ERROR - get_data(%s): expected %d, but nil", id, key, expected[idx])
			} else {
				if value.V != expected[idx] {
					t.Errorf("[%d] ERROR - get_data(%s):%v!=%v", id, key, value.V, expected[idx])
				}
			}
		}
	}
}

// without cache on err
func TestDataCacheMap(t *testing.T) {
	l := newScriptedLoader[int](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     scriptedLoadData(l),
		Expiration:   20,
		EvictTimeout: 60000,
		WaitTimeout:  1,
	})
	get := func() { cache.Get("1") }
	tryGetDataMap(t, cache, 1, 0, "1", nil)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1}
	pollUntil(t, "load", func() bool { return cache.Get("1") != nil })
	tryGetDataMap(t, cache, 2, 0, "1", 1)
	tryGetDataMap(t, cache, 3, 20, "1", 1)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 2}
	pollUntil(t, "reload", func() bool { return cache.Get("1").V != 1 })
	tryGetDataMap(t, cache, 4, 0, "1", 2)

	l = newScriptedLoader[int](t)
	cache = NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     scriptedLoadData(l),
		Expiration:   20,
		EvictTimeout: 60000,
		WaitTimeout:  1,
	})
	tryGetDataMap(t, cache, 8, 0, "3", nil)
	l.waitStart(t, func() { cache.Get("3") })
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", func() bool {
		_, err := cache.GetWithError("3")
		return errors.Is(err, errTestLoad)
	})
	tryGetDataMap(t, cache, 9, 0, "3", nil) // if get nil, indicate data is not get from cache since loadData not return nil
}

func TestBatchDataCacheMap(t *testing.T) {
	l := newScriptedLoader[int](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadDataBatch: scriptedLoadDataBatch(l),
		Expiration:    20,
		EvictTimeout:  60000,
		RetryInterval: 10,
		WaitTimeout:   1,
	})
	get := func() { cache.GetMany([]string{"1", "2"}) }

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil})
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1}
	pollUntil(t, "load", getManyReturns(cache, []string{"1", "2"}, 1))
	tryGetManyDataMap(t, cache, 2, 0, []string{"1", "2"}, []interface{}{1, 1})
	tryGetManyDataMap(t, cache, 3, 20, []string{"1", "2"}, []interface{}{1, 1})
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 2}
	pollUntil(t, "reload", getManyReturns(cache, []string{"1", "2"}, 2))
	tryGetManyDataMap(t, cache, 4, 0, []string{"1", "2"}, []interface{}{2, 2})

	// return err when not cache on err (indicate data will not be cached when err without cache on err)
	l = newScriptedLoader[int](t)
	cache = NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     scriptedLoadData(l),
		Expiration:   20,
		EvictTimeout: 60000,
		WaitTimeout:  1,
	})
	tryGetDataMap(t, cache, 1, 0, "1", nil)
	l.waitStart(t, func() { cache.Get("1") })
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", func() bool {
		_, err := cache.GetWithError("1")
		return errors.Is(err, errTestLoad)
	})
	tryGetDataMap(t, cache, 2, 0, "1", nil)
}

func TestBatchDataCacheMapBySingleLoad(t *testing.T) {
	l := newScriptedLoader[int](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:      scriptedLoadData(l),
		Expiration:    20,
		EvictTimeout:  60000,
		RetryInterval: 60000,
		WaitTimeout:   1,
	})
	complete := func(value int, keys ...string) {
		t.Helper()
		for range keys {
			l.waitStart(t, func() { cache.GetMany(keys) })
			l.results <- loadResult[int]{value: value}
		}
		pollUntil(t, "loads", getManyReturns(cache, keys, value))
	}

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil})
	complete(1, "1", "2")
	tryGetManyDataMap(t, cache, 2, 0, []string{"1", "2"}, []interface{}{1, 1})
	tryGetManyDataMap(t, cache, 3, 20, []string{"1", "2"}, []interface{}{1, 1})
	complete(2, "1", "2")
	tryGetManyDataMap(t, cache, 4, 0, []string{"1", "2"}, []interface{}{2, 2})

	tryGetDataMap(t, cache, 5, 0, "5", nil)
	tryGetDataMap(t, cache, 6, 0, "6", nil)
	complete(1, "5", "6")
	tryGetManyDataMap(t, cache, 7, 0, []string{"5", "6"}, []interface{}{1, 1})
	tryGetManyDataMap(t, cache, 8, 20, []string{"5", "6"}, []interface{}{1, 1})
	complete(2, "5", "6")
	tryGetDataMap(t, cache, 9, 0, "5", 2)
	tryGetDataMap(t, cache, 10, 0, "6", 2)

	// return err when not cache on err (indicate data will not be cached when err without cache on err)
	l = newScriptedLoader[int](t)
	cache = NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     scriptedLoadData(l),
		Expiration:   20,
		EvictTimeout: 60000,
		WaitTimeout:  1,
	})
	tryGetDataMap(t, cache, 11, 0, "1", nil)
	l.waitStart(t, func() { cache.Get("1") })
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", func() bool {
		_, err := cache.GetWithError("1")
		return errors.Is(err, errTestLoad)
	})
	tryGetDataMap(t, cache, 12, 0, "1", nil)
}

func TestErrDataCacheMap(t *testing.T) {
	l := newScriptedLoader[int](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:        scriptedLoadData(l),
		Expiration:      20,
		ExpirationOnErr: 10,
		EvictTimeout:    60000,
		WaitTimeout:     1,
	})
	get := func() { cache.Get("1") }
	tryGetDataMap(t, cache, 1, 0, "1", nil)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", func() bool { return cache.Get("1") != nil })
	tryGetDataMap(t, cache, 2, 0, "1", 1)
	tryGetDataMap(t, cache, 3, 20, "1", 1)
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 2, err: errTestLoad}
	pollUntil(t, "failed reload", func() bool { return cache.Get("1").V != 1 })
	tryGetDataMap(t, cache, 4, 0, "1", 2)
}

// cache on err, and data in dataMap returned by load func not nil
func TestBatchDataCacheMapErr(t *testing.T) {
	l := newScriptedLoader[int](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadDataBatch:   scriptedLoadDataBatch(l),
		Expiration:      20,
		ExpirationOnErr: 20,
		EvictTimeout:    60000,
		RetryInterval:   10,
		WaitTimeout:     1,
	})
	get := func() { cache.GetMany([]string{"1", "2"}) }

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil})
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 1, err: errTestLoad}
	pollUntil(t, "failed load", getManyReturns(cache, []string{"1", "2"}, 1))
	tryGetManyDataMap(t, cache, 2, 0, []string{"1", "2"}, []interface{}{1, 1})
	tryGetManyDataMap(t, cache, 3, 20, []string{"1", "2"}, []interface{}{1, 1})
	l.waitStart(t, get)
	l.results <- loadResult[int]{value: 2, err: errTestLoad}
	pollUntil(t, "failed reload", getManyReturns(cache, []string{"1", "2"}, 2))
	tryGetManyDataMap(t, cache, 4, 0, []string{"1", "2"}, []interface{}{2, 2})
}

// cache on err, and data in dataMap returned by load func nil
// test will it panic if loadDataBatch func return nil with error
func TestBatchNilDataCacheMapErr(t *testing.T) {
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadDataBatch:   sleepGetBatchNilDataMapErr,
		Expiration:      20,
		ExpirationOnErr: 20,
		EvictTimeout:    30,
		RetryInterval:   10,
		WaitTimeout:     10000,
	})

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil})
}

func TestCacheMapGetManyConcurrent(t *testing.T) {
	cases := []struct {
		name  string
		batch bool
	}{
		{name: "LoadData"},
		{name: "LoadDataBatch", batch: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var loads atomic.Int64
			opt := &CacheMapOption[string, *testData]{
				LoadData: func(key string) (*testData, error) {
					return &testData{K: key, V: int(loads.Add(1))}, nil
				},
				Expiration:    1,
				RetryInterval: 1,
			}
			if tc.batch {
				opt.LoadDataBatch = func(keys []string) (map[string]*testData, error) {
					v := int(loads.Add(1))
					data := make(map[string]*testData)
					for _, key := range keys {
						data[key] = &testData{K: key, V: v}
					}
					return data, nil
				}
			}
			cache := NewCacheMap(opt)
			keys := []string{"a", "b", "c", "d"}
			check := func(key string, value *testData) {
				if value != nil && (value.K != key || value.V <= 0) {
					t.Errorf("value of %q = %+v, want nil or a successful load", key, value)
				}
			}
			deadline := time.Now().Add(200 * time.Millisecond)
			var wg sync.WaitGroup
			for g := 0; g < 8; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; time.Now().Before(deadline); i++ {
						if g%2 == 0 {
							key := keys[(g+i)%len(keys)]
							check(key, cache.Get(key))
							continue
						}
						for key, value := range cache.GetMany(keys) {
							check(key, value)
						}
					}
				}()
			}
			wg.Wait()
		})
	}
}

func TestCacheMapGetWithError(t *testing.T) {
	found := &testData{K: "k", V: 1}
	errorTimeValue := &testData{K: "k", V: -1}
	cases := []struct {
		name            string
		value           *testData
		err             error
		block           bool
		expirationOnErr int64
		waitTimeout     int64
		waitsFull       bool
		wantValue       *testData
		wantErr         errorWant
		wantGet         *testData
	}{
		{name: "found", value: found, waitTimeout: 10000, wantValue: found, wantErr: wantNoError, wantGet: found},
		{name: "not found", waitTimeout: 10000, wantErr: wantNoError},
		{name: "first load failed, retry pending", value: errorTimeValue, err: errTestLoad, waitTimeout: 300, waitsFull: true, wantErr: wantLoadFailed},
		{name: "first load failed with ExpirationOnErr", value: errorTimeValue, err: errTestLoad, expirationOnErr: 60000, waitTimeout: 10000, wantErr: wantLoadFailed, wantGet: errorTimeValue},
		{name: "first load not finished within WaitTimeout", block: true, waitTimeout: 20, waitsFull: true, wantErr: wantNotLoaded},
		{name: "WaitTimeout 0", block: true, wantErr: wantNotLoaded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			cache := NewCacheMap(&CacheMapOption[string, *testData]{
				LoadData: func(string) (*testData, error) {
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
			value, err := cache.GetWithError("k")
			if elapsed := time.Since(start); tc.waitsFull && elapsed < time.Duration(tc.waitTimeout)*time.Millisecond {
				t.Errorf("GetWithError returned after %v, before WaitTimeout %dms", elapsed, tc.waitTimeout)
			}
			if value != tc.wantValue {
				t.Errorf("value = %+v, want %+v", value, tc.wantValue)
			}
			checkError(t, err, tc.wantErr)
			if got := cache.Get("k"); got != tc.wantGet {
				t.Errorf("Get = %+v, want %+v", got, tc.wantGet)
			}
		})
	}
}

func TestCacheMapGetWithErrorAfterGetMany(t *testing.T) {
	found := &testData{K: "k", V: 1}
	errorTimeValue := &testData{K: "k", V: -1}
	cases := []struct {
		name            string
		batch           map[string]*testData
		err             error
		expirationOnErr int64
		wantValue       *testData
		wantErr         errorWant
	}{
		{name: "found", batch: map[string]*testData{"k": found}, wantValue: found, wantErr: wantNoError},
		{name: "not found", batch: map[string]*testData{}, wantErr: wantNoError},
		{name: "failed", err: errTestLoad, wantErr: wantLoadFailed},
		{name: "failed with ExpirationOnErr", batch: map[string]*testData{"k": errorTimeValue}, err: errTestLoad, expirationOnErr: 60000, wantErr: wantLoadFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cache := NewCacheMap(&CacheMapOption[string, *testData]{
				LoadData: func(key string) (*testData, error) {
					t.Errorf("unexpected LoadData(%q)", key)
					return nil, nil
				},
				LoadDataBatch: func([]string) (map[string]*testData, error) {
					return tc.batch, tc.err
				},
				Expiration:      60000,
				ExpirationOnErr: tc.expirationOnErr,
				RetryInterval:   60000,
				WaitTimeout:     300,
			})
			cache.GetMany([]string{"k"})
			value, err := cache.GetWithError("k")
			if value != tc.wantValue {
				t.Errorf("value = %+v, want %+v", value, tc.wantValue)
			}
			checkError(t, err, tc.wantErr)
		})
	}
}

func TestCacheMapGetWithErrorRetryRunning(t *testing.T) {
	l := newScriptedLoader[*testData](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:      func(string) (*testData, error) { return l.load() },
		Expiration:    60000,
		RetryInterval: 100,
	})
	get := func() { cache.GetWithError("k") }
	l.waitStart(t, get)
	l.results <- loadResult[*testData]{value: &testData{K: "k", V: -1}, err: errTestLoad}
	pollUntil(t, "first load failure", func() bool {
		_, err := cache.GetWithError("k")
		return errors.Is(err, errTestLoad)
	})

	l.waitStart(t, get)
	value, err := cache.GetWithError("k")
	if value != nil {
		t.Errorf("value while retry running = %+v, want nil", value)
	}
	checkError(t, err, wantLoadFailed)

	found := &testData{K: "k", V: 1}
	l.results <- loadResult[*testData]{value: found}
	pollUntil(t, "retry success", func() bool {
		value, err := cache.GetWithError("k")
		return value == found && err == nil
	})
}

func TestCacheMapGetWithErrorRefreshFailed(t *testing.T) {
	l := newScriptedLoader[*testData](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:      func(string) (*testData, error) { return l.load() },
		Expiration:    1,
		EvictTimeout:  60000,
		RetryInterval: 60000,
	})
	get := func() { cache.GetWithError("k") }
	found := &testData{K: "k", V: 1}
	l.waitStart(t, get)
	l.finish(t, found, nil, get)

	l.results <- loadResult[*testData]{value: &testData{K: "k", V: -1}, err: errTestLoad}
	for i := 0; i < 50; i++ {
		value, err := cache.GetWithError("k")
		if value != found || err != nil {
			t.Fatalf("GetWithError after failed refresh = (%+v, %v), want (%+v, nil)", value, err, found)
		}
		time.Sleep(time.Millisecond)
	}
	if got := cache.Get("k"); got != found {
		t.Errorf("Get = %+v, want %+v", got, found)
	}
}

func TestCacheMapGetWithErrorExpirationOnErr(t *testing.T) {
	l := newScriptedLoader[*testData](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:        func(string) (*testData, error) { return l.load() },
		Expiration:      1,
		ExpirationOnErr: 1,
		EvictTimeout:    60000,
		RetryInterval:   60000,
	})
	get := func() { cache.GetWithError("k") }
	first := &testData{K: "k", V: 1}
	l.waitStart(t, get)
	l.finish(t, first, nil, get)
	value, err := cache.GetWithError("k")
	if value != first {
		t.Errorf("value after success = %+v, want %+v", value, first)
	}
	checkError(t, err, wantNoError)

	errorTimeValue := &testData{K: "k", V: -1}
	l.finish(t, errorTimeValue, errTestLoad, get)
	value, err = cache.GetWithError("k")
	if value != nil {
		t.Errorf("value after failure = %+v, want nil", value)
	}
	checkError(t, err, wantLoadFailed)
	if got := cache.Get("k"); got != errorTimeValue {
		t.Errorf("Get after failure = %+v, want %+v", got, errorTimeValue)
	}

	second := &testData{K: "k", V: 2}
	l.finish(t, second, nil, get)
	value, err = cache.GetWithError("k")
	if value != second {
		t.Errorf("value after recovery = %+v, want %+v", value, second)
	}
	checkError(t, err, wantNoError)
}

func TestCacheMapGetWithErrorEvicted(t *testing.T) {
	l := newScriptedLoader[*testData](t)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:      func(string) (*testData, error) { return l.load() },
		Expiration:    5,
		EvictTimeout:  20,
		RetryInterval: 60000,
	})
	get := func() { cache.GetWithError("k") }
	l.waitStart(t, get)
	l.results <- loadResult[*testData]{err: errTestLoad}
	pollUntil(t, "first load failure", func() bool {
		_, err := cache.GetWithError("k")
		return errors.Is(err, errTestLoad)
	})
	pollUntil(t, "eviction", func() bool {
		cache.mu.RLock()
		defer cache.mu.RUnlock()
		_, ok := cache.dataMap["k"]
		return !ok
	})

	value, err := cache.GetWithError("k")
	if value != nil {
		t.Errorf("value after eviction = %+v, want nil", value)
	}
	checkError(t, err, wantNotLoaded)

	found := &testData{K: "k", V: 1}
	l.waitStart(t, get)
	l.results <- loadResult[*testData]{value: found}
	pollUntil(t, "load after eviction", func() bool {
		value, err := cache.GetWithError("k")
		return value == found && err == nil
	})
}

func TestCacheMapGetWithErrorConcurrent(t *testing.T) {
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
			cache := NewCacheMap(&CacheMapOption[string, *testData]{
				LoadData: func(key string) (*testData, error) {
					n := loads.Add(1)
					if n%2 == 0 {
						return &testData{K: key, V: -1}, errTestLoad
					}
					return &testData{K: key, V: int(n)}, nil
				},
				Expiration:      1,
				ExpirationOnErr: tc.expirationOnErr,
				RetryInterval:   1,
			})
			keys := []string{"a", "b", "c", "d"}
			deadline := time.Now().Add(200 * time.Millisecond)
			var wg sync.WaitGroup
			for g := 0; g < 8; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; time.Now().Before(deadline); i++ {
						key := keys[(g+i)%len(keys)]
						value, err := cache.GetWithError(key)
						if err != nil {
							if value != nil || !errors.Is(err, ErrNotLoaded) {
								t.Errorf("GetWithError(%q) = (%+v, %v), want nil and ErrNotLoaded", key, value, err)
							}
						} else if value == nil || value.K != key || value.V <= 0 {
							t.Errorf("GetWithError(%q) = (%+v, nil), want a successful load", key, value)
						}
					}
				}()
			}
			wg.Wait()
		})
	}
}

func BenchmarkDataCache_Get(b *testing.B) {
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:      sleepGetDataSimple,
		Expiration:    20,
		EvictTimeout:  30,
		RetryInterval: 10,
		WaitTimeout:   10,
	})
	keys := []string{"1", "2", "3", "4", "5"}
	var finish sync.WaitGroup
	for i := 0; i < len(keys); i++ {
		finish.Add(1)
		go func(index int) {
			for j := 0; j < b.N; j++ {
				cache.Get(keys[index])
			}
			finish.Done()
		}(i)
	}
	finish.Wait()
}

func BenchmarkDataCache_GetMany(b *testing.B) {
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:      sleepGetDataSimple,
		LoadDataBatch: sleepGetManyDataSimple,
		Expiration:    20,
		EvictTimeout:  30,
		RetryInterval: 10,
		WaitTimeout:   10,
	})
	keys := [][]string{{"1"}, {"2", "3"}, {"1", "3"}, {"4"}, {"2", "4", "5"}}
	var finish sync.WaitGroup
	for i := 0; i < len(keys); i++ {
		finish.Add(1)
		go func(index int) {
			for j := 0; j < b.N; j++ {
				cache.GetMany(keys[index])
			}
			finish.Done()
		}(i)
	}
	finish.Wait()
}
