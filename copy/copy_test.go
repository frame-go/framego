package copy

import (
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

type basicStruct struct {
	A int
}

type testStruct struct {
	A int
	B []int
	C map[string]int
	D *basicStruct
}

type simpleTestStruct struct {
	C map[string]int
	D *basicStruct
}

type embeddedTestStruct struct {
	testStruct
	E string
}

// Copy from category_api to avoid circular import
type CategoryBrief struct {
	DisplayName        *string `json:"display_name"`
	Catid              *int32  `json:"catid"`
	Image              *string `json:"image"`
	NoSub              bool    `json:"no_sub"`
	IsDefaultSubcat    *int32  `json:"is_default_subcat"`
	BlockBuyerPlatform []int32 `json:"block_buyer_platform"`
}

type copyFunc func(dst interface{}, src interface{}) error

var copyFuncs = []copyFunc{
	JsonDeepCopy,
	GobDeepCopy,
	MsgpackDeepCopy,
	JsoniterDeepCopy,
	ffjsonDeepCopy,
	shamatonMsgpackDeepCopy,
}

var categoryList []*CategoryBrief

func getFunctionName(f copyFunc) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func TryCopyBasic(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)
	srcInt, dstInt := 1, 2
	errInt := f(&dstInt, srcInt)
	if errInt != nil {
		t.Errorf("[%s]Deep copy error %v", fName, errInt)
	}
	if srcInt != dstInt {
		t.Errorf("[%s]Expected %v, but %v", fName, srcInt, dstInt)
	}
	dstBool := false
	errBool := f(&dstBool, true)
	if errBool != nil {
		t.Errorf("[%s]Deep copy error %v", fName, errBool)
	}
	if !dstBool {
		t.Errorf("[%s]Expected %v, but %v", fName, true, dstBool)
	}
	srcString, dstString := "hello", "world"
	errString := f(&dstString, srcString)
	if errString != nil {
		t.Errorf("[%s]Deep copy error %v", fName, errString)
	}
	if srcString != dstString {
		t.Errorf("[%s]Expected %v, but %v", fName, dstString, dstInt)
	}
	srcArray, dstArray := [1]int{1}, [1]int{2}
	errArray := f(&dstArray, srcArray)
	if errArray != nil {
		t.Errorf("[%s]Deep copy error %v", fName, errArray)
	}
	if srcArray != dstArray {
		t.Errorf("[%s]Expected %v, but %v", fName, dstArray, dstInt)
	}
}

func TryCopyStruct(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)
	src := testStruct{
		A: 1,
		B: []int{1, 2, 3},
		C: map[string]int{
			"hello": 1,
		},
		D: &basicStruct{
			A: 1,
		},
	}
	dst := testStruct{}
	err := f(&dst, &src)
	if err != nil {
		t.Errorf("[%s]Deep copy error %v", fName, err)
	}
	src.A = 2
	if dst.A != 1 {
		t.Errorf("[%s]Expected %v, but %v", fName, 1, dst.A)
	}
	src.B = append(src.B, 4)
	if len(dst.B) != 3 {
		t.Errorf("[%s]Expected %v, but %v", fName, 3, len(dst.B))
	}
	src.C["hello"] = 2
	if dst.C["hello"] != 1 {
		t.Errorf("[%s]Expected %v, but %v", fName, 1, dst.C["hello"])
	}
	src.D.A = 2
	if dst.D.A != 1 {
		t.Errorf("[%s]Expected %v, but %v", fName, 1, dst.D.A)
	}

	// test copy to another struct type
	simpleDst := simpleTestStruct{}
	src.C["hello"] = 1
	src.D.A = 1
	err = f(&simpleDst, &src)
	src.C["hello"] = 2
	if simpleDst.C["hello"] != 1 {
		t.Errorf("[%s]Expected %v, but %v", fName, 1, simpleDst.C["hello"])
	}
	src.D.A = 2
	if simpleDst.D.A != 1 {
		t.Errorf("[%s]Expected %v, but %v", fName, 1, simpleDst.D.A)
	}
}

func TryCopyMap(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)
	src := map[string][]int{
		"hello": {1, 2, 3},
	}
	dst := make(map[string][]int)
	err := f(&dst, &src)
	if err != nil {
		t.Errorf("[%s]Deep copy error %v", fName, err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Errorf("[%s]Expected deepequal, but not", fName)
	}
	src["hello"] = append(src["hello"], 4)
	if len(dst["hello"]) != 3 {
		t.Errorf("[%s]Expected %v, but %v", fName, 3, len(dst["hello"]))
	}
}

func TryCopySlice(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)
	src := []*basicStruct{
		{
			A: 1,
		},
	}
	dst := make([]*basicStruct, 0)
	err := f(&dst, &src)
	if err != nil {
		t.Errorf("[%s]Deep copy error %v", fName, err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Errorf("[%s]Expected deepequal, but not", fName)
	}
	src[0].A = 2
	if dst[0].A != 1 {
		t.Errorf("[%s]Expected %v, but %v", fName, 1, dst[0].A)
	}
	src = append(src, &basicStruct{
		A: 2,
	})
	if len(dst) != 1 {
		t.Errorf("[%s]Expected %v, but %v", fName, 1, len(dst))
	}
}

func TryCopyEmptyInterface(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)
	var tests = []struct {
		dst interface{}
		src interface{}
	}{
		{int64(1), math.MinInt64},
		{false, true},
		{"zhao pengcheng", "zhao pengjie"},
		{[1]int{1}, [1]int{2}},
	}

	for _, test := range tests {
		dst, src := test.dst, test.src
		if err := f(&dst, src); err != nil {
			t.Errorf("[%s]Deep copy error %v", fName, err)
		}
		if src != dst {
			t.Errorf("[%s]Expected %t, but %t", fName, src, dst)
		}
	}
}

func TryCopyConcurrently(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)
	var wg sync.WaitGroup
	for i := 0; i < 500000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var src, dst int
			src = rand.Int()
			f(&dst, src)
			if src != dst {
				t.Errorf("[%s]Expected %v, but %v", fName, src, dst)
			}
		}()
	}
}

func TryCopyRealData(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)
	dst := make([]*CategoryBrief, 0)
	_ = f(&dst, categoryList)
	if len(categoryList) != len(dst) {
		t.Errorf("[%s]Failed to deep copy real data", fName)
	}
}

func TryCopyWithFiltering(t *testing.T, f copyFunc) {
	fName := getFunctionName(f)

	src := embeddedTestStruct{
		testStruct: testStruct{
			A: 4,
			B: []int{1, 2, 3},
			C: map[string]int{},
			D: nil,
		},
		E: "test",
	}
	dst := testStruct{}
	_ = f(&dst, src)
	if dst.A != src.A {
		t.Errorf("[%s]Failed to deep copy with filtering", fName)
	}
}

func TestDeepCopy(t *testing.T) {
	for _, f := range copyFuncs {
		TryCopyBasic(t, f)
		TryCopyMap(t, f)
		TryCopySlice(t, f)
		TryCopyStruct(t, f)
		TryCopyConcurrently(t, f)
		TryCopyRealData(t, f)
		//TryCopyWithFiltering(t, f)
		//TryCopyEmptyInterface(t, f)
	}
}

func CopyStruct(f copyFunc) {
	src := testStruct{
		A: 1,
		B: []int{1, 2, 3},
		C: map[string]int{
			"hello": 1,
		},
		D: &basicStruct{
			A: 1,
		},
	}
	dst := testStruct{}
	f(&dst, src)
}

func CopyMap(f copyFunc) {
	src := map[string][]int{
		"hello": {1, 2, 3},
	}
	dst := make(map[string][]int)
	f(&dst, src)
}

func CopySlice(f copyFunc) {
	src := []*basicStruct{
		{
			A: 1,
		},
	}
	dst := make([]*basicStruct, 0)
	f(&dst, src)
}

func CopyRealData(f copyFunc) {
	dst := make([]*CategoryBrief, 0)
	f(&dst, categoryList)
}

func benchmarkDeepCopyFakeData(f copyFunc, n int) {
	for i := 0; i < n; i++ {
		CopyMap(f)
		CopySlice(f)
		CopyStruct(f)
	}
}

func benchmarkDeepCopyRealData(f copyFunc, n int) {
	for i := 0; i < n; i++ {
		CopyRealData(f)
	}
}

func BenchmarkJSONDeepCopyFakeData(b *testing.B) {
	benchmarkDeepCopyFakeData(JsonDeepCopy, b.N)
}

func BenchmarkJSONDeepCopyRealData(b *testing.B) {
	benchmarkDeepCopyRealData(JsonDeepCopy, b.N)
}

func BenchmarkGOBDeepCopyFakeData(b *testing.B) {
	benchmarkDeepCopyFakeData(GobDeepCopy, b.N)
}

func BenchmarkGOBDeepCopyRealData(b *testing.B) {
	benchmarkDeepCopyRealData(GobDeepCopy, b.N)
}

func BenchmarkMsgpackDeepCopyFakeData(b *testing.B) {
	benchmarkDeepCopyFakeData(MsgpackDeepCopy, b.N)
}

func BenchmarkMsgpackDeepCopyRealData(b *testing.B) {
	benchmarkDeepCopyRealData(MsgpackDeepCopy, b.N)
}

func BenchmarkJsoniterDeepCopyFakeData(b *testing.B) {
	benchmarkDeepCopyFakeData(JsoniterDeepCopy, b.N)
}

func BenchmarkJsoniterDeepCopyRealData(b *testing.B) {
	benchmarkDeepCopyRealData(JsoniterDeepCopy, b.N)
}

func BenchmarkFfjsonDeepCopyFakeData(b *testing.B) {
	benchmarkDeepCopyFakeData(ffjsonDeepCopy, b.N)
}

func BenchmarkFfjsonDeepCopyRealData(b *testing.B) {
	benchmarkDeepCopyRealData(ffjsonDeepCopy, b.N)
}

func BenchmarkShamatonMsgpackDeepCopyFakeData(b *testing.B) {
	benchmarkDeepCopyFakeData(shamatonMsgpackDeepCopy, b.N)
}

func BenchmarkShamatonMsgpackDeepCopyRealData(b *testing.B) {
	benchmarkDeepCopyRealData(shamatonMsgpackDeepCopy, b.N)
}

func TestMain(m *testing.M) {
	_ = json.Unmarshal([]byte(`[{"display_name":"ios_automation_search_category","catid":15759,"image":"dc99cb9103743b8eef16f2612325920e","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"ios_automation_cateogry_multiple_banners","catid":15820,"image":"dc99cb9103743b8eef16f2612325920e","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"ios_automation_category_single_Banner","catid":15765,"image":"dc99cb9103743b8eef16f2612325920e","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Mobile \u0026 Gadgets","catid":8,"image":"487cae0dd1782a67ef30a47c23f263fa","no_sub":false,"is_default_subcat":0,"block_buyer_platform":[1]},{"display_name":"automationTestCategory","catid":15752,"image":"4ec0c285b5ad5eefc2c916ed03143b85","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Toys, Kids \u0026 Babies","catid":12,"image":"e3536d5f67fc99649709fe7be594b3c7","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Mobile \u0026 Accessories","catid":12935,"image":"60e4dec7ca5bd841d9f394ad32b8ac6a","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"cosmetic\u0026fragrances","catid":6,"image":"3a3b3abdba2cd95b6f2ed1e62e659bb2","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Computers \u0026 Peripherals","catid":9,"image":"880e39b4f91a7744c360b5b19c80b449","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"testcat1","catid":7691,"image":"50d7b03a82bb3b36e83b48d5d95a6536","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":7692,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"Men's Apparel","catid":2,"image":"0a2015bebc468bac5b990923741c79b4","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"                                                   Bags","catid":3,"image":"f79aa9ca1d5e2be8c4d9e88bfa19ec5d","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Women's Shoes","catid":126,"image":"9bcc1af6d1ac96c01a30bdd3ceff30c3","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"automationTestCategory","catid":15740,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Men's Shoes","catid":171,"image":"d64d494d34a94582fef7c23030e2a1e6","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Sports Shoes","catid":15310,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Watches","catid":5,"image":"f81aece708ef00af408f8cbfb129165f","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Sports \u0026 Hobbies","catid":13,"image":"874b27f541e146411dc803b09b0fc807","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Design \u0026 Crafts","catid":10,"image":"18a6cd242f35696d208627e8fdf99c52","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Pet Accessories","catid":166,"image":"7c0d2a7646168059e95a6935cfaf4ef5","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Tickets \u0026 Vouchers","catid":175,"image":"78b7752e2fd7cf7b7844e21f56fafd04","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Miscellaneous ","catid":15,"image":"ae2263a98be949fc9bd81c4af61af5ad","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"books","catid":177,"image":"a656f2846832636699a8e0181fd24197","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"试一下可以吗？喵喵","catid":4,"image":"89bcba92f77983cdc274267e9c09a51f","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Bangles","catid":640,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"dresses","catid":1709,"image":"9ba29c310b17e9c612015cfb78fca3d2","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"อุปกรณ์ถ่ายภาพและเครื่องเสียง","catid":7710,"image":"a656f2846832636699a8e0181fd24197","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":7711,"image":"8af8a3351856c55634eb720c5828a782","no_sub":false,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"Testcat2😁😂😃😄 ","catid":7694,"image":"a656f2846832636699a8e0181fd24197","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":14574,"image":"5d6073167e59c45c2a2df80041a8b98b","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"Non-Default","catid":7697,"image":"639391726c26278a15d147454fd64c47","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":7695,"image":"639391726c26278a15d147454fd64c47","no_sub":false,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"home appliances","catid":1590,"image":"a656f2846832636699a8e0181fd24197","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Kitchen Appliances","catid":1592,"image":"95db00a597e5252c93f0392dd6e9e869","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"deleting !!!!!","catid":14457,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"Home \u0026 Living","catid":11,"image":"49c12265d42358677f1eac1dd6943258","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"sasasas","catid":13016,"image":"3e4f6f246a3c6c4b86a826411620eb6e","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":14395,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":7771,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"New Default for Amulya","catid":14643,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":14557,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"Default","catid":14555,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"Default","catid":14553,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"dave","catid":14719,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"test l1 qqq","catid":14701,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":14705,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"test one category","catid":14836,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Default","catid":14838,"image":"","no_sub":true,"is_default_subcat":1,"block_buyer_platform":null},{"display_name":"core_autopass","catid":14833,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"rachel_test_mandatory_mall ","catid":14924,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"CBBlock1","catid":14895,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"","catid":14884,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"FRP test","catid":14881,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"yuhangtestlongcategorynamewithoutwhitespaceatbradnlistpagelalalalalalalaalalalalalalalalalalalalalallaallalalallalalalalallalalalalalaalalalalallalalalallalalallalalallaalallalalalbiubiubiubiubiubiubiubiubiubiuyuhangyuhangyuhangyuhangendofname","catid":15126,"image":"7817d6dbf213a73f58aff6d655007d7c","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"TEST THREE LEVELS","catid":14970,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"TEST TWO LEVELS","catid":14967,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"TEST ONE LEVEL","catid":14964,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"yuhang test long name with  white space at brand list page lalalalalalala","catid":15129,"image":"4da682a54bc85a699b81df415cbc712b","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"ECHO","catid":15159,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Camping Tools","catid":15198,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":" bức thư đã vượt qua biên giới dưới cùng của thanh chữ cái dính","catid":15153,"image":"1939c7e51978925571e0498e186df06e","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"white space at the end of name                                                                   ","catid":15150,"image":"00884b2d08a48ab59b4ff4e2e5ce2ee3","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"                      white space at the start of name","catid":15147,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"long long longllllllllllllllllllllllllllllllllllllllllllllllllllmmmmmmm name with spaces","catid":15144,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"สุดท้ายก็กลับบ้าน","catid":15184,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"\u003cinput\u003e","catid":15141,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Test Category Filter","catid":15240,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"thisisalongname lalalallalalallalllalalalalalallalalallaallalallalallalalalallalallalallaallaalalallaallalalalalla  endofname","catid":15181,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"test refactoring","catid":15394,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"图书","catid":15320,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"testssss","catid":15344,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"onlyL1Cat","catid":15341,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"sytestnew","catid":15335,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"xtest","catid":15479,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"","catid":15487,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"TestL1Attribute","catid":15484,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"laowangL1.2","catid":15532,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"LAO WANG","catid":15527,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"cheese_test_1","catid":15599,"image":"","no_sub":false,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"","catid":15756,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"Baby Products","catid":15753,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"test L3 cat","catid":15718,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"sip测试","catid":15870,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"test11","catid":15886,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"m_sgcategory3","catid":16045,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"m_sgcategory3","catid":16044,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"mits l1","catid":16103,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null},{"display_name":"L1[1]L2[1]L3[1]L4[3]L5[1]","catid":16137,"image":"","no_sub":true,"is_default_subcat":0,"block_buyer_platform":null}]`), &categoryList)
	retCode := m.Run()
	os.Exit(retCode)
}
