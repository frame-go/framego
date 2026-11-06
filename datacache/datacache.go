package datacache

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/frame-go/framego/copy"
)

// LoadDataCallback a call back function used to load data from remote.
type LoadDataCallback[T any] func() (T, error)

// DataCache a thread/goroutine safe in memory cache storage.
type DataCache[T any] struct {
	data            *T
	loaded          bool  // data holds init data or a stored load result
	loadErr         error // error of the latest load, kept only while data is not from a successful load
	loadData        LoadDataCallback[T]
	expiration      int64 // in nanoseconds
	expirationOnErr int64 // in nanoseconds
	retryInterval   int64 // in nanoseconds
	waitTimeout     int64 // in nanoseconds
	initCutOffTime  int64 // unix timestamp in nanoseconds
	nextUpdateTime  int64 // unix timestamp in nanoseconds
	mu              sync.RWMutex
}

// CacheInitOption initiation params of DataCache
type CacheInitOption[T any] struct {
	WithInitData    bool
	InitData        T
	LoadData        LoadDataCallback[T]
	Expiration      int64 // data expiration, in milliseconds
	ExpirationOnErr int64 // data expiration when err occur, will not cache if not set, in milliseconds
	RetryInterval   int64 // retry interval for load data if failed, in milliseconds. If 0, use 1 second
	WaitTimeout     int64 // waiting timeout for first data, in milliseconds
}

const (
	// MilliToNanoSecond millisecond to nanosecond ratio
	MilliToNanoSecond = int64(time.Millisecond / time.Nanosecond)

	// DefaultRetryInterval default retry interval
	DefaultRetryInterval int64 = 1000000000
)

// ErrNotLoaded is wrapped in the error GetWithError returns when it has no data from init data or a successful load to return
var ErrNotLoaded = errors.New("datacache_not_loaded")

var errLoadInProgress = fmt.Errorf("%w: load_in_progress", ErrNotLoaded)

// NewDataCache create new DataCache with a CacheInitOption
// The data updating is synchronized, the Get method will return old data while updating in progress
// Only one go routine will try to load the data during one RetryInterval
// If didn't provide init data, first Get method call will trigger data loading,
// and all Get requests will wait for first data until WaitTimeout
func NewDataCache[T any](opt *CacheInitOption[T]) (c *DataCache[T]) {
	c = &DataCache[T]{
		data:            &opt.InitData,
		loadData:        opt.LoadData,
		expiration:      opt.Expiration * MilliToNanoSecond,
		expirationOnErr: opt.ExpirationOnErr * MilliToNanoSecond,
		retryInterval:   opt.RetryInterval * MilliToNanoSecond,
		waitTimeout:     opt.WaitTimeout * MilliToNanoSecond,
		nextUpdateTime:  0,
	}
	if c.retryInterval <= 0 {
		c.retryInterval = DefaultRetryInterval
	}
	if c.waitTimeout <= 0 {
		c.waitTimeout = 0
	}
	if opt.WithInitData || !reflect.ValueOf(opt.InitData).IsZero() {
		c.loaded = true
		c.initCutOffTime = time.Now().UnixNano()
	}
	return
}

// Get data from DataCache
// Return zero value if no data is loaded yet, or the value the loader returned if the latest load failed with ExpirationOnErr set,
// use GetWithError to tell these from data of a successful load
// Note the returned data is reference, any modification in returned data will affect the subsequent returned data
func (c *DataCache[T]) Get() T {
	c.waitData()
	c.mu.RLock()
	defer c.mu.RUnlock()
	return *c.data
}

// GetWithError gets data from DataCache, waiting for first data like Get
// Return zero value and an error wrapping ErrNotLoaded if there is no init data and no load has succeeded yet,
// or if the latest load failed with ExpirationOnErr set; the error also wraps the load error if there is one
// Note the returned data is reference, any modification in returned data will affect the subsequent returned data
func (c *DataCache[T]) GetWithError() (T, error) {
	c.waitData()
	c.mu.RLock()
	defer c.mu.RUnlock()
	var zeroValue T
	if c.loadErr != nil {
		return zeroValue, fmt.Errorf("%w: %w", ErrNotLoaded, c.loadErr)
	}
	if !c.loaded {
		return zeroValue, errLoadInProgress
	}
	return *c.data, nil
}

// waitData starts a load if data expired or retry needed, and waits for first data until WaitTimeout
func (c *DataCache[T]) waitData() {
	now := time.Now().UnixNano()
	c.mu.RLock()
	nextUpdateTime := c.nextUpdateTime
	c.mu.RUnlock()
	if now >= nextUpdateTime {
		// data expired, start go routine to update
		doUpdate := false
		c.mu.Lock()
		if now >= c.nextUpdateTime {
			c.nextUpdateTime = now + c.retryInterval
			doUpdate = true
		}
		if c.initCutOffTime == 0 {
			c.initCutOffTime = now + c.waitTimeout
		}
		c.mu.Unlock()
		if doUpdate {
			go func() {
				var data T
				var err error
				data, err = c.loadData()
				expire := c.expiration
				if err != nil {
					if c.expirationOnErr <= 0 {
						c.mu.Lock()
						if !c.loaded {
							c.loadErr = err
						}
						c.mu.Unlock()
						return
					}
					expire = c.expirationOnErr
				}
				now := time.Now().UnixNano()
				c.mu.Lock()
				c.data = &data
				c.loaded = true
				c.loadErr = err
				c.nextUpdateTime = now + expire
				if now < c.initCutOffTime {
					c.initCutOffTime = now
				}
				c.mu.Unlock()
			}()
		}
	}

	for {
		c.mu.RLock()
		initCutOffTime := c.initCutOffTime
		c.mu.RUnlock()
		if now > initCutOffTime {
			break
		}
		time.Sleep(time.Millisecond)
		now += MilliToNanoSecond
	}
}

// GetCopy get deep copy of data from DataCache
func (c *DataCache[T]) GetCopy(v T) error {
	data := c.Get()
	err := copy.DeepCopy(v, data)
	return err
}
