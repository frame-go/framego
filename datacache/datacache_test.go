package datacache

import (
	"errors"
	"os"
	"testing"
	"time"
)

var counter = 0

func sleepGetData() (int, error) {
	time.Sleep(10 * time.Millisecond)
	counter++
	return counter, nil
}

func sleepGetErrData() (int, error) {
	time.Sleep(10 * time.Millisecond)
	counter++
	return counter, errors.New("dummy")
}

func sleepGetErrDataNil() (*int, error) {
	time.Sleep(10 * time.Millisecond)
	counter++
	return nil, errors.New("dummy")
}

func sleepGetDataPtr() (*int, error) {
	time.Sleep(10 * time.Millisecond)
	counter++
	value := counter
	return &value, nil
}

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

func TestDataCache(t *testing.T) {
	// with init data
	counter = 0
	cache := NewDataCache(&CacheInitOption[int]{
		InitData:   0,
		LoadData:   sleepGetData,
		Expiration: 50,
	})
	tryGetData(t, cache, 1, 0, 0)
	tryGetData(t, cache, 2, 30, 1)
	tryGetData(t, cache, 3, 70, 1)
	tryGetData(t, cache, 4, 30, 2)

	// without init data
	counter = 0
	cache = NewDataCache(&CacheInitOption[int]{
		LoadData:    sleepGetData,
		Expiration:  20,
		WaitTimeout: 1,
	})
	tryGetData(t, cache, 5, 0, 0)
	tryGetData(t, cache, 6, 30, 1)

	// without init data (loadData* func return err without cache on err)
	counter = 0
	cache = NewDataCache(&CacheInitOption[int]{
		LoadData:    sleepGetErrData,
		Expiration:  20,
		WaitTimeout: 1,
	})
	tryGetData(t, cache, 7, 0, 0)
	tryGetData(t, cache, 8, 30, 0)
	if counter != 2 {
		t.Errorf("counter should be 2 (seelpGetErr* func got invocked 2 times), but is %v", counter)
	}

	// without init data (loadData* func return err with cache on err)
	counter = 0
	cache = NewDataCache(&CacheInitOption[int]{
		InitData:        0,
		LoadData:        sleepGetErrData,
		Expiration:      50,
		ExpirationOnErr: 40,
	})
	tryGetData(t, cache, 9, 0, 0)
	tryGetData(t, cache, 10, 30, 1)
	tryGetData(t, cache, 11, 70, 1)
	tryGetData(t, cache, 12, 30, 2)

	// without init data (loadData* func return err with cache on err)
	counter = 0
	cache = NewDataCache(&CacheInitOption[int]{
		LoadData:        sleepGetErrData,
		Expiration:      20,
		ExpirationOnErr: 10,
		WaitTimeout:     1,
	})
	tryGetData(t, cache, 13, 0, 0)
	tryGetData(t, cache, 14, 30, 1)
	time.Sleep(time.Second)
}

func TestDataNilCache(t *testing.T) {
	// without init data (loadData* func return err and nil as to cache data with cache on err)
	counter = 0
	cache := NewDataCache(&CacheInitOption[*int]{
		LoadData:        sleepGetErrDataNil,
		Expiration:      20,
		ExpirationOnErr: 20,
		WaitTimeout:     1,
	})
	tryGetDataPtr(t, cache, 15, 0, nil)
	tryGetDataPtr(t, cache, 16, 10, nil)
	if counter != 1 {
		t.Errorf("counter should be 1 (seelpGetErr* func got invocked 1 time), but is %v", counter)
	}
	time.Sleep(time.Second)
}

func TestDataPtrCache(t *testing.T) {
	// with init data
	counter = 0
	cache := NewDataCache(&CacheInitOption[*int]{
		InitData:   nil,
		LoadData:   sleepGetDataPtr,
		Expiration: 50,
	})
	tryGetDataPtr(t, cache, 1, 0, nil)
	tryGetDataPtr(t, cache, 2, 30, 1)
	tryGetDataPtr(t, cache, 3, 70, 1)
	tryGetDataPtr(t, cache, 4, 30, 2)
}

func TestMain(m *testing.M) {
	if os.Getenv("CI") == "" {
		m.Run()
	}
}
