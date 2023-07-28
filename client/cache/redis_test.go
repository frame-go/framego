package cache

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/frame-go/framego/errors"
)

type TestStruct struct {
	IntValue int
	StrValue string
}

func assertError(t *testing.T, err error, msg string, a ...any) {
	if err != nil {
		t.Errorf("[Error: %v] %s", err, fmt.Sprintf(msg, a...))
	}
}

func assertCondition(t *testing.T, assert bool, msg string, a ...any) {
	if !assert {
		t.Errorf("Assert error: %s", fmt.Sprintf(msg, a...))
	}
}

func newClient(t *testing.T) Client {
	c, err := NewRedisClient(&Config{
		Address: "127.0.0.1:6379",
	})
	if err != nil {
		t.Error(err)
	}
	return c
}

func getTestData() map[string]any {
	return map[string]any{
		"testInt32":   int(-100),
		"testUint32":  uint32(3000000000),
		"testInt64":   int64(-40000000000000),
		"testUint64":  uint64(10446744073709551616),
		"testFloat32": float32(3.1415),
		"testFloat64": float64(3.141592653),
		"testString":  "String",
		"testBytes":   []byte{1, 2, 3, 4},
		"testStruct":  TestStruct{1, "test"},
	}
}

func getTestDataStorage() map[string]any {
	var vInt32 int
	var vUint32 uint32
	var vInt64 int64
	var vUint64 uint64
	var vFloat32 float32
	var vFloat64 float64
	var vString string
	var vBytes []byte
	var vStruct TestStruct
	return map[string]any{
		"testInt32":   &vInt32,
		"testUint32":  &vUint32,
		"testInt64":   &vInt64,
		"testUint64":  &vUint64,
		"testFloat32": &vFloat32,
		"testFloat64": &vFloat64,
		"testString":  &vString,
		"testBytes":   &vBytes,
		"testStruct":  &vStruct,
	}
}

func TestGetSet(t *testing.T) {
	// init
	c := newClient(t)
	ctx := context.Background()
	testData := getTestData()
	keys := []string{}
	for k := range testData {
		keys = append(keys, k)
		_, err := c.Delete(ctx, k)
		assertError(t, err, "step 0: clean %s", k)
	}
	exists, err := c.Exists(ctx, keys...)
	assertError(t, err, "step 0: clean check exist")
	assertCondition(t, exists == 0, "step 0: clean check exist %d > 0", exists)
	testDataStorage := getTestDataStorage()
	for k := range testDataStorage {
		err := c.Get(ctx, k, testDataStorage[k])
		assertCondition(t, errors.Is(err, Nil), "step 0: get %s error %s != Nil", k, err)
	}

	// set
	for k, v := range testData {
		err := c.Set(ctx, k, v, 0)
		assertError(t, err, "step 1: set %s", k)
	}

	// get
	exists, err = c.Exists(ctx, keys...)
	assertError(t, err, "step 1: clean check exist")
	assertCondition(t, exists == len(keys), "step 1: clean check exist %d != %d", exists, len(keys))
	for k, v := range testData {
		err := c.Get(ctx, k, testDataStorage[k])
		assertError(t, err, "step 2: get %s", k)
		equal := reflect.DeepEqual(v, reflect.ValueOf(testDataStorage[k]).Elem().Interface())
		assertCondition(t, equal, "step2: not equal %s, %v != %v", k, v, reflect.ValueOf(testDataStorage[k]).Elem())
	}

	// delete
	deleted, err := c.Delete(ctx, keys...)
	assertError(t, err, "step 3: delete")
	assertCondition(t, deleted == len(keys), "step 3: delete %v != %v", deleted, len(keys))
	exists, err = c.Exists(ctx, keys...)
	assertError(t, err, "step 3: clean check exist")
	assertCondition(t, exists == 0, "step 3: clean check exist %d > 0", exists)
}

func TestGetSetStructPointer(t *testing.T) {
	// init
	c := newClient(t)
	ctx := context.Background()
	key := "testStruct"
	value := &TestStruct{1, "test"}
	_, err := c.Delete(ctx, key)
	assertError(t, err, "step 0: clean")

	// set
	err = c.Set(ctx, key, value, 0)
	assertError(t, err, "step 1: set")

	// get by pointer to pointer
	valueGet := &TestStruct{}
	err = c.Get(ctx, key, &valueGet)
	assertError(t, err, "step 2: get")
	assertCondition(t, reflect.DeepEqual(valueGet, value), "step 2: get %v != %v", valueGet, value)

	// get by pointer
	valueGet = &TestStruct{}
	err = c.Get(ctx, key, valueGet)
	assertError(t, err, "step 3: get")
	assertCondition(t, reflect.DeepEqual(valueGet, value), "step 3: get %v != %v", valueGet, value)
}

func TestMultiGetSet(t *testing.T) {
	// init
	c := newClient(t)
	ctx := context.Background()
	testData := getTestData()
	keys := []string{}
	for k := range testData {
		keys = append(keys, k)
	}
	_, err := c.Delete(ctx, keys...)
	assertError(t, err, "step 0: clean")

	// mset
	err = c.MSet(ctx, testData)
	assertError(t, err, "step 1: mset")

	// mget
	testDataStorage := getTestDataStorage()
	testNilDataStorage := getTestDataStorage()
	for _, k := range keys {
		testDataStorage[k+"Nil"] = testNilDataStorage[k]
	}
	err = c.MGet(ctx, testDataStorage)
	assertError(t, err, "step 2: mget")
	for _, k := range keys {
		v := testData[k]
		equal := reflect.DeepEqual(v, reflect.ValueOf(testDataStorage[k]).Elem().Interface())
		assertCondition(t, equal, "step 2: not equal %s, %v != %v", k, v, reflect.ValueOf(testDataStorage[k]).Elem())
		kNil := k + "Nil"
		vNil := testDataStorage[kNil]
		assertCondition(t, vNil == nil, "step 2: not nil %s, %v != nil", kNil, vNil)
	}
}

func TestAddUpdate(t *testing.T) {
	// init
	c := newClient(t)
	ctx := context.Background()
	key := "test"
	var value string
	value0 := "value0"
	value1 := "value1"
	_, err := c.Delete(ctx, key)
	assertError(t, err, "step 0: clean")

	// update and add
	ok, err := c.Update(ctx, key, value0, 0)
	assertError(t, err, "step 1: update")
	assertCondition(t, !ok, "step 1: update ok")
	exists, err := c.Exists(ctx, key)
	assertError(t, err, "step 1: update exists")
	assertCondition(t, exists == 0, "step 1: update exists %d", exists)
	ok, err = c.Add(ctx, key, value0, 0)
	assertError(t, err, "step 1: add")
	assertCondition(t, ok, "step 1: add ok")
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 1: add exists")
	assertCondition(t, exists == 1, "step 1: add exists %d", exists)

	// add and update
	ok, err = c.Add(ctx, key, value1, 0)
	assertError(t, err, "step 2: add")
	assertCondition(t, !ok, "step 2: add ok")
	err = c.Get(ctx, key, &value)
	assertError(t, err, "step 2: add get")
	assertCondition(t, value == value0, "step 2: add get not equal %v != %v", value, value0)
	ok, err = c.Update(ctx, key, value1, 0)
	assertError(t, err, "step 2: update")
	assertCondition(t, ok, "step 2: update ok")
	err = c.Get(ctx, key, &value)
	assertError(t, err, "step 2: update get")
	assertCondition(t, value == value1, "step 2: update get not equal %v != %v", value, value1)
}

func TestExpiration(t *testing.T) {
	// init
	c := newClient(t)
	ctx := context.Background()
	key := "test"
	value := "value"
	_, err := c.Delete(ctx, key)
	assertError(t, err, "step 0: clean")

	// set with expiry
	err = c.Set(ctx, key, value, 100*time.Millisecond)
	assertError(t, err, "step 1: set")
	exists, err := c.Exists(ctx, key)
	assertError(t, err, "step 1: set exists")
	assertCondition(t, exists == 1, "step 1: set exists %d", exists)
	time.Sleep(200 * time.Millisecond)
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 1: check exists")
	assertCondition(t, exists == 0, "step 1: check exists %d", exists)

	// update with expiry
	err = c.Set(ctx, key, value, 100*time.Millisecond)
	assertError(t, err, "step 2: set")
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 2: set exists")
	assertCondition(t, exists == 1, "step 2: set exists %d", exists)
	ok, err := c.Update(ctx, key, value, 300*time.Millisecond)
	assertError(t, err, "step 2: update")
	assertCondition(t, ok, "step 2: update ok")
	time.Sleep(200 * time.Millisecond)
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 2: check exists")
	assertCondition(t, exists == 1, "step 2: check exists %d", exists)

	// keep expiry
	ok, err = c.Update(ctx, key, value, KeepExpiration)
	assertError(t, err, "step 3: update")
	assertCondition(t, ok, "step 3: update ok")
	time.Sleep(200 * time.Millisecond)
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 3: check exists")
	assertCondition(t, exists == 0, "step 3: check exists %d", exists)

	// update expiry
	err = c.Set(ctx, key, value, 100*time.Millisecond)
	assertError(t, err, "step 4: set")
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 4: set exists")
	assertCondition(t, exists == 1, "step 4: set exists %d", exists)
	ok, err = c.Expire(ctx, key, 1*time.Second)
	assertError(t, err, "step 4: expire")
	assertCondition(t, ok, "step 4: expire ok")
	time.Sleep(200 * time.Millisecond)
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 4: check exists")
	assertCondition(t, exists == 1, "step 4: check exists %d", exists)
	time.Sleep(1 * time.Second)
	exists, err = c.Exists(ctx, key)
	assertError(t, err, "step 4: check expired")
	assertCondition(t, exists == 0, "step 4: check expired %d", exists)
}

func TestIncrBy(t *testing.T) {
	// init
	c := newClient(t)
	ctx := context.Background()
	key := "test"
	var value int64 = 100
	var valueGet int64 = 0
	_, err := c.Delete(ctx, key)
	assertError(t, err, "step 0: clean")

	// set
	err = c.Set(ctx, key, value, 10*time.Millisecond)
	assertError(t, err, "step 1: set")
	err = c.Get(ctx, key, &valueGet)
	assertError(t, err, "step 1: get")
	assertCondition(t, valueGet == value, "step 1: get %v != %v", valueGet, value)

	// incr
	valueGet, err = c.IncrBy(ctx, key, 10)
	valueExpect := value + 10
	assertError(t, err, "step 2: incr")
	assertCondition(t, valueGet == valueExpect, "step 2: incr %v != %v", valueGet, valueExpect)
	err = c.Get(ctx, key, &valueGet)
	assertError(t, err, "step 2: get")
	assertCondition(t, valueGet == valueExpect, "step 2: get %v != %v", valueGet, valueExpect)

	// decr
	valueGet, err = c.IncrBy(ctx, key, -200)
	valueExpect = valueExpect - 200
	assertError(t, err, "step 3: decr")
	assertCondition(t, valueGet == valueExpect, "step 3: decr %v != %v", valueGet, valueExpect)
	err = c.Get(ctx, key, &valueGet)
	assertError(t, err, "step 3: get")
	assertCondition(t, valueGet == valueExpect, "step 3: get %v != %v", valueGet, valueExpect)

	// incr from empty value
	deleted, err := c.Delete(ctx, key)
	assertError(t, err, "step 4: delete")
	assertCondition(t, deleted == 1, "step 4: delete %v != 1", deleted)
	valueGet, err = c.IncrBy(ctx, key, value)
	assertError(t, err, "step 4: incr")
	assertCondition(t, valueGet == value, "step 4: incr %v != %v", valueGet, value)
	err = c.Get(ctx, key, &valueGet)
	assertError(t, err, "step 4: get")
	assertCondition(t, valueGet == value, "step 4: get %v != %v", valueGet, value)
}

func TestMain(m *testing.M) {
	if os.Getenv("CI") == "" {
		m.Run()
	}
}
