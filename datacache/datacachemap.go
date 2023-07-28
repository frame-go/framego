package datacache

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/frame-go/framego/copy"
)

// LoadDataByKeyCallback is a call back to load data from source
type LoadDataByKeyCallback[K comparable, V any] func(key K) (V, error)

// LoadDataBatchByKeysCallback is a call back to load batch data from source
type LoadDataBatchByKeysCallback[K comparable, V any] func(keys []K) (map[K]V, error)

// CacheItem is basic unit of cache
type CacheItem[V any] struct {
	data           *V
	nextUpdateTime int64 // unix timestamp in nanoseconds
	initCutOffTime int64 // unix timestamp in nanoseconds
	muItemUpdate   sync.RWMutex
	evictTime      int64 // in nanosecond
	muItemEvict    sync.RWMutex
}

// CacheMap is a collection for CacheItems
type CacheMap[K comparable, V any] struct {
	dataMap         map[K]*CacheItem[V]
	loadData        LoadDataByKeyCallback[K, V]
	loadDataBatch   LoadDataBatchByKeysCallback[K, V]
	expiration      int64 // in nanoseconds
	expirationOnErr int64 // in nanoseconds
	retryInterval   int64 // in nanoseconds
	waitTimeout     int64 // in nanoseconds
	evictTimeout    int64 // in nanoseconds
	mu              sync.RWMutex
	evictTicker     *time.Ticker
}

// CacheMapOption has all options of a CacheMap
type CacheMapOption[K comparable, V any] struct {
	LoadData        LoadDataByKeyCallback[K, V]
	LoadDataBatch   LoadDataBatchByKeysCallback[K, V] // optional, if not provided, fallback to LoadData
	Expiration      int64                             // data update interval, in milliseconds
	ExpirationOnErr int64                             // data update interval when err occur, will not cache if not set
	EvictTimeout    int64                             // timeout for evict data from cache, in milliseconds. If 0, use Expiration * 2
	RetryInterval   int64                             // retry interval for load data if failed, in milliseconds. If 0, use 1 second
	WaitTimeout     int64                             // waiting timeout for first data, in milliseconds
}

// NewCacheMap creates new data cache by DataCacheMapOption
// The data updating is async, the Get method will return old data while updating in progress
// Only one goroutine will try to load the data during one RetryInterval
// First Get method call will trigger data loading, all Get requests will wait for first data until WaitTimeout
// The data will be loaded again if exceed Expiration since last load,
// and will be removed from cache if exceed EvictTimeout since last access.
func NewCacheMap[K comparable, V any](opt *CacheMapOption[K, V]) (c *CacheMap[K, V]) {
	c = &CacheMap[K, V]{
		loadData:        opt.LoadData,
		loadDataBatch:   opt.LoadDataBatch,
		dataMap:         make(map[K]*CacheItem[V]),
		expiration:      opt.Expiration * MilliToNanoSecond,
		expirationOnErr: opt.ExpirationOnErr * MilliToNanoSecond,
		evictTimeout:    opt.EvictTimeout * MilliToNanoSecond,
		retryInterval:   opt.RetryInterval * MilliToNanoSecond,
		waitTimeout:     opt.WaitTimeout * MilliToNanoSecond,
	}
	if c.evictTimeout <= c.expiration {
		c.evictTimeout = c.expiration * 2
	}
	if c.retryInterval <= 0 {
		c.retryInterval = DefaultRetryInterval
	}
	if c.waitTimeout <= 0 {
		c.waitTimeout = 0
	}

	// Start background goroutine for GC.
	c.evictTicker = time.NewTicker(time.Duration(c.evictTimeout))
	go func() {
		for {
			<-c.evictTicker.C
			c.evictKeys()
		}
	}()

	return
}

func (c *CacheMap[K, V]) updateData(key K, item *CacheItem[V]) {
	var data V
	var err error
	data, err = c.loadData(key)
	expire := c.expiration
	if err != nil {
		if c.expirationOnErr <= 0 {
			return
		}
		expire = c.expirationOnErr
	}
	now := time.Now().UnixNano()
	item.muItemUpdate.Lock()
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&item.data)), unsafe.Pointer(&data))
	item.nextUpdateTime = now + expire
	if now < item.initCutOffTime {
		item.initCutOffTime = now
	}
	item.muItemUpdate.Unlock()
}

// GC
func (c *CacheMap[K, V]) evictKeys() {
	now := time.Now().UnixNano()
	c.mu.RLock()
	var keys []K
	for k, v := range c.dataMap {
		v.muItemEvict.RLock()
		if v.evictTime < now {
			keys = append(keys, k)
		}
		v.muItemEvict.RUnlock()
	}
	c.mu.RUnlock()
	c.mu.Lock()
	for _, k := range keys {
		delete(c.dataMap, k)
	}
	c.mu.Unlock()
}

// Get gets data by key from CacheMap
// Return zero value if no valid data
// Note the returned data is reference, any modification in returned data will affect the subsequent returned data
func (c *CacheMap[K, V]) Get(key K) V {
	now := time.Now().UnixNano()
	var item *CacheItem[V]
	ok := false

	c.mu.RLock()
	item, ok = c.dataMap[key]
	c.mu.RUnlock()

	if !ok {
		c.mu.Lock()
		item, ok = c.dataMap[key]
		if !ok {
			item = &CacheItem[V]{
				nextUpdateTime: now + c.retryInterval,
				evictTime:      now + c.evictTimeout,
				initCutOffTime: now + c.waitTimeout,
			}
			c.dataMap[key] = item
		}
		c.mu.Unlock()

		// No existing data, start go routine to fetch data
		if !ok {
			go c.updateData(key, item)
		}
	}

	// Update data if data expired or retry needed
	item.muItemUpdate.RLock()
	nextUpdateTime := item.nextUpdateTime
	item.muItemUpdate.RUnlock()
	if now > nextUpdateTime {
		// Guarantee only 1 goroutine created for update
		item.muItemUpdate.Lock()
		if now > item.nextUpdateTime {
			item.nextUpdateTime = now + c.retryInterval
			go c.updateData(key, item)
		}
		item.muItemUpdate.Unlock()
	}

	// If didn't load init data yet, wait for data until cut off time
	for {
		item.muItemUpdate.RLock()
		initCutOffTime := item.initCutOffTime
		item.muItemUpdate.RUnlock()
		if now > initCutOffTime {
			break
		}
		time.Sleep(time.Millisecond)
		now += MilliToNanoSecond
	}

	// Update data evict time, guarantee it uses biggest value of all goroutines.
	c.updateEvictTime(item, now)

	// This could be zero value if remote data is not available or error happens, or simply waitTimeout.
	item.muItemUpdate.RLock()
	defer item.muItemUpdate.RUnlock()
	if item.data == nil {
		var zeroValue V
		return zeroValue
	}
	return *item.data
}

// GetCopy gets deep copy of data by key from CacheMap
func (c *CacheMap[K, V]) GetCopy(key K, v V) error {
	data := c.Get(key)
	err := copy.DeepCopy(v, data)
	return err
}

func (c *CacheMap[K, V]) updateEvictTime(item *CacheItem[V], now int64) {
	item.muItemEvict.Lock()
	newEvictTime := now + c.evictTimeout
	if newEvictTime > item.evictTime {
		item.evictTime = newEvictTime
	}
	item.muItemEvict.Unlock()
}

func (c *CacheMap[K, V]) updateBatch(unCachedDataMap map[K]*CacheItem[V], updatedStatus *CacheItem[V]) {
	uncachedKeys := make([]K, 0, len(unCachedDataMap))
	for key := range unCachedDataMap {
		uncachedKeys = append(uncachedKeys, key)
	}
	var data map[K]V
	var err error
	data, err = c.loadDataBatch(uncachedKeys)
	expire := c.expiration
	if err != nil {
		if c.expirationOnErr <= 0 {
			return
		}
		expire = c.expirationOnErr
	}
	now := time.Now().UnixNano()
	for key, item := range unCachedDataMap {
		itemData := data[key]
		item.muItemUpdate.Lock()
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&item.data)), unsafe.Pointer(&itemData))
		item.nextUpdateTime = now + expire
		if now < item.initCutOffTime {
			item.initCutOffTime = now
		}
		item.muItemUpdate.Unlock()
	}
	updatedStatus.muItemUpdate.Lock()
	updatedStatus.initCutOffTime = now
	updatedStatus.muItemUpdate.Unlock()
}

// GetMany gets data by key from CacheMap
// Return zero value if no valid data
// Note the returned data is reference, any modification in returned data will affect the subsequent returned data
func (c *CacheMap[K, V]) GetMany(keys []K) map[K]V {
	now := time.Now().UnixNano()
	data := make(map[K]V, len(keys))

	var uncachedKeys []K

	c.mu.RLock()
	for _, key := range keys {
		item, ok := c.dataMap[key]
		if ok {
			c.updateEvictTime(item, now)

			item.muItemUpdate.RLock()
			if item.data == nil {
				var zeroValue V
				data[key] = zeroValue
			} else {
				data[key] = *item.data
			}
			if now > item.nextUpdateTime {
				uncachedKeys = append(uncachedKeys, key)
			}
			item.muItemUpdate.RUnlock()
		} else {
			uncachedKeys = append(uncachedKeys, key)
		}
	}
	c.mu.RUnlock()

	if len(uncachedKeys) > 0 {
		unCachedDataMap := make(map[K]*CacheItem[V], len(uncachedKeys))
		hasNonExistKey := false
		c.mu.Lock()
		for _, uncachedKey := range uncachedKeys {
			item, ok := c.dataMap[uncachedKey]
			if !ok {
				item = &CacheItem[V]{
					nextUpdateTime: now + c.retryInterval,
					evictTime:      now + c.evictTimeout,
					initCutOffTime: now + c.waitTimeout,
				}
				c.dataMap[uncachedKey] = item
				hasNonExistKey = true
				unCachedDataMap[uncachedKey] = item
			} else if now > item.nextUpdateTime {
				item.nextUpdateTime = now + c.retryInterval
				unCachedDataMap[uncachedKey] = item
			}
		}
		c.mu.Unlock()

		if len(unCachedDataMap) > 0 {
			updatedStatus := &CacheItem[V]{
				initCutOffTime: now + c.waitTimeout,
			}
			if c.loadDataBatch == nil {
				for key, item := range unCachedDataMap {
					go c.updateData(key, item)
				}
			} else {
				go c.updateBatch(unCachedDataMap, updatedStatus)
			}

			// If didn't load init data yet, wait for data until cut off time
			if hasNonExistKey {
				for {
					updatedStatus.muItemUpdate.RLock()
					initCutOffTime := updatedStatus.initCutOffTime
					updatedStatus.muItemUpdate.RUnlock()
					if now > initCutOffTime {
						break
					}
					time.Sleep(time.Millisecond)
					now += MilliToNanoSecond
				}
			}
		}

		for key, item := range unCachedDataMap {
			item.muItemUpdate.RLock()
			if item.data == nil {
				var zeroValue V
				data[key] = zeroValue
			} else {
				data[key] = *item.data
			}
			item.muItemUpdate.RUnlock()
		}
	}

	return data
}
