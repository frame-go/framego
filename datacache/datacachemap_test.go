package datacache

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type testData struct {
	K string
	V int
}

var counterMap map[string]*testData
var muMap map[string]*sync.Mutex
var mu sync.Mutex

func sleepGetDataMap(key string) (*testData, error) {
	mu.Lock()
	defer mu.Unlock()
	d, ok := counterMap[key]
	if !ok {
		counterMap[key] = &testData{K: key, V: 0}
		d = counterMap[key]
	}
	time.Sleep(10 * time.Millisecond)
	d.V++
	return d, nil
}

func sleepGetDataMapPerKey(key string) (*testData, error) {
	if _, ok := muMap[key]; !ok {
		muMap[key] = &sync.Mutex{}
	}
	muKey := muMap[key]
	muKey.Lock()
	defer muKey.Unlock()
	d, ok := counterMap[key]
	if !ok {
		counterMap[key] = &testData{K: key, V: 0}
		d = counterMap[key]
	}
	time.Sleep(10 * time.Millisecond)
	d.V++
	return d, nil
}

var sleepGetDataMapErrCount = 0

func sleepGetDataMapErr(key string) (*testData, error) {
	mu.Lock()
	defer mu.Unlock()
	sleepGetDataMapErrCount++
	d, ok := counterMap[key]
	if !ok {
		counterMap[key] = &testData{K: key, V: 0}
		d = counterMap[key]
	}
	time.Sleep(10 * time.Millisecond)
	d.V++
	return d, errors.New("dummy")
}

func sleepGetNilDataMapErr(key string) (*testData, error) {
	time.Sleep(10 * time.Millisecond)
	return nil, errors.New("dummy")
}

func sleepGetBatchDataMap(keys []string) (map[string]*testData, error) {
	mu.Lock()
	defer mu.Unlock()
	data := make(map[string]*testData)
	for _, key := range keys {
		d, ok := counterMap[key]
		if !ok {
			counterMap[key] = &testData{K: key, V: 0}
			d = counterMap[key]
		}
		d.V++
		data[key] = d
	}
	time.Sleep(10 * time.Millisecond)
	return data, nil
}

func sleepGetBatchDataMapErr(keys []string) (map[string]*testData, error) {
	mu.Lock()
	defer mu.Unlock()
	data := make(map[string]*testData)
	for _, key := range keys {
		d, ok := counterMap[key]
		if !ok {
			counterMap[key] = &testData{K: key, V: 0}
			d = counterMap[key]
		}
		d.V++
		data[key] = d
	}
	time.Sleep(10 * time.Millisecond)
	return data, errors.New("dummy")
}

var errCounterWG sync.WaitGroup

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
	counterMap = make(map[string]*testData)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     sleepGetDataMap,
		Expiration:   20,
		EvictTimeout: 30,
		WaitTimeout:  1,
	})
	tryGetDataMap(t, cache, 1, 0, "1", nil) // 5
	tryGetDataMap(t, cache, 2, 15, "1", 1)  // 15
	tryGetDataMap(t, cache, 3, 20, "1", 1)  // 35
	tryGetDataMap(t, cache, 4, 15, "1", 2)  // 50

	counterMap = make(map[string]*testData)
	cache = NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     sleepGetDataMapErr,
		Expiration:   20,
		EvictTimeout: 30,
		WaitTimeout:  1,
	})
	tryGetDataMap(t, cache, 8, 0, "3", nil)
	tryGetDataMap(t, cache, 9, 10, "3", nil) // if get nil, indicate data is not get from cache since loadData not return nil
	time.Sleep(time.Second)
}

func TestBatchDataCacheMap(t *testing.T) {
	counterMap = make(map[string]*testData)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadDataBatch: sleepGetBatchDataMap,
		Expiration:    20,
		EvictTimeout:  30,
		RetryInterval: 10,
		WaitTimeout:   1,
	})

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil}) // 5
	tryGetManyDataMap(t, cache, 2, 15, []string{"1", "2"}, []interface{}{1, 1})    // 15
	tryGetManyDataMap(t, cache, 3, 20, []string{"1", "2"}, []interface{}{1, 1})    // 35
	tryGetManyDataMap(t, cache, 4, 15, []string{"1", "2"}, []interface{}{2, 2})    // 50

	// return err when not cache on err (indicate data will not be cached when err without cache on err)
	cache = NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     sleepGetDataMapErr,
		Expiration:   20,
		EvictTimeout: 30,
		WaitTimeout:  1,
	})
	tryGetDataMap(t, cache, 1, 0, "1", nil)  // 5
	tryGetDataMap(t, cache, 2, 15, "1", nil) // 15
	time.Sleep(time.Second)
}

func TestBatchDataCacheMapBySingleLoad(t *testing.T) {
	counterMap = make(map[string]*testData)
	muMap = make(map[string]*sync.Mutex)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:      sleepGetDataMapPerKey,
		Expiration:    20,
		EvictTimeout:  30,
		RetryInterval: 10,
		WaitTimeout:   1,
	})

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil}) // 5
	tryGetManyDataMap(t, cache, 2, 15, []string{"1", "2"}, []interface{}{1, 1})    // 15
	tryGetManyDataMap(t, cache, 3, 20, []string{"1", "2"}, []interface{}{1, 1})    // 35
	tryGetManyDataMap(t, cache, 4, 15, []string{"1", "2"}, []interface{}{2, 2})    // 50

	tryGetDataMap(t, cache, 5, 0, "5", nil)
	tryGetDataMap(t, cache, 6, 0, "6", nil)
	tryGetManyDataMap(t, cache, 7, 15, []string{"5", "6"}, []interface{}{1, 1}) // 15
	tryGetManyDataMap(t, cache, 8, 20, []string{"5", "6"}, []interface{}{1, 1}) // 35
	tryGetDataMap(t, cache, 9, 20, "5", 2)                                      // 55
	tryGetDataMap(t, cache, 10, 0, "6", 2)                                      // 55

	// return err when not cache on err (indicate data will not be cached when err without cache on err)
	cache = NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:     sleepGetDataMapErr,
		Expiration:   20,
		EvictTimeout: 30,
		WaitTimeout:  1,
	})
	tryGetDataMap(t, cache, 11, 0, "1", nil)  // 5
	tryGetDataMap(t, cache, 12, 15, "1", nil) // 15
	time.Sleep(time.Second)
}

func TestErrDataCacheMap(t *testing.T) {
	counterMap = make(map[string]*testData)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadData:        sleepGetDataMapErr,
		Expiration:      20,
		ExpirationOnErr: 10,
		EvictTimeout:    30,
		WaitTimeout:     1,
	})
	tryGetDataMap(t, cache, 1, 0, "1", nil) // 5
	tryGetDataMap(t, cache, 2, 15, "1", 1)  // 15
	tryGetDataMap(t, cache, 3, 20, "1", 1)  // 35
	tryGetDataMap(t, cache, 4, 15, "1", 2)  // 50
	time.Sleep(time.Second)
}

// cache on err, and data in dataMap returned by load func not nil
func TestBatchDataCacheMapErr(t *testing.T) {
	counterMap = make(map[string]*testData)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadDataBatch:   sleepGetBatchDataMapErr,
		Expiration:      20,
		ExpirationOnErr: 20,
		EvictTimeout:    30,
		RetryInterval:   10,
		WaitTimeout:     1,
	})

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil}) // 5
	tryGetManyDataMap(t, cache, 2, 15, []string{"1", "2"}, []interface{}{1, 1})    // 15
	tryGetManyDataMap(t, cache, 3, 20, []string{"1", "2"}, []interface{}{1, 1})    // 35
	tryGetManyDataMap(t, cache, 4, 15, []string{"1", "2"}, []interface{}{2, 2})    // 50
	time.Sleep(time.Second)
}

// cache on err, and data in dataMap returned by load func nil
// test will it panic if loadDataBatch func return nil with error
func TestBatchNilDataCacheMapErr(t *testing.T) {
	counterMap = make(map[string]*testData)
	cache := NewCacheMap(&CacheMapOption[string, *testData]{
		LoadDataBatch:   sleepGetBatchNilDataMapErr,
		Expiration:      20,
		ExpirationOnErr: 20,
		EvictTimeout:    30,
		RetryInterval:   10,
		WaitTimeout:     1,
	})

	tryGetManyDataMap(t, cache, 1, 0, []string{"1", "2"}, []interface{}{nil, nil}) // 5
	time.Sleep(time.Second)
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
