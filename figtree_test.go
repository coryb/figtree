package figtree

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"testing"

	yaml "gopkg.in/coryb/yaml.v2"
	logging "gopkg.in/op/go-logging.v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type info struct {
	name string
	line string
}

func line() string {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Sprintf("%s:%d", path.Base(file), line)
}

func init() {
	StringifyValue = false
	logging.SetLevel(logging.NOTICE, "")
}

func newFigTreeFromEnv(opts ...Option) *FigTree {
	cwd, _ := os.Getwd()
	opts = append([]Option{
		WithHome(os.Getenv("HOME")),
		WithCwd(cwd),
		WithEnvPrefix("FIGTREE"),
	}, opts...)

	return NewFigTree(opts...)
}

type TestOptions struct {
	String1    StringOption     `json:"str1,omitempty" yaml:"str1,omitempty"`
	LeaveEmpty StringOption     `json:"leave-empty,omitempty" yaml:"leave-empty,omitempty"`
	Array1     ListStringOption `json:"arr1,omitempty" yaml:"arr1,omitempty"`
	Map1       MapStringOption  `json:"map1,omitempty" yaml:"map1,omitempty"`
	Int1       IntOption        `json:"int1,omitempty" yaml:"int1,omitempty"`
	Float1     Float32Option    `json:"float1,omitempty" yaml:"float1,omitempty"`
	Bool1      BoolOption       `json:"bool1,omitempty" yaml:"bool1,omitempty"`
}

type TestBuiltin struct {
	String1    string            `yaml:"str1,omitempty"`
	LeaveEmpty string            `yaml:"leave-empty,omitempty"`
	Array1     []string          `yaml:"arr1,omitempty"`
	Map1       map[string]string `yaml:"map1,omitempty"`
	Int1       int               `yaml:"int1,omitempty"`
	Float1     float32           `yaml:"float1,omitempty"`
	Bool1      bool              `yaml:"bool1,omitempty"`
}

func TestOptionsLoadConfigD3(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d3arr1val1"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d3arr1val2"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "dupval"})
	arr1 = append(arr1, StringOption{"../figtree.yml", true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{"../figtree.yml", true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{"../../figtree.yml", true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{"../../figtree.yml", true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{"figtree.yml", true, "d3str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": StringOption{"../../figtree.yml", true, "d1map1val0"},
			"key1": StringOption{"../figtree.yml", true, "d2map1val1"},
			"key2": StringOption{"figtree.yml", true, "d3map1val2"},
			"key3": StringOption{"figtree.yml", true, "d3map1val3"},
			"dup":  StringOption{"figtree.yml", true, "d3dupval"},
		},
		Int1:   IntOption{"figtree.yml", true, 333},
		Float1: Float32Option{"figtree.yml", true, 3.33},
		Bool1:  BoolOption{"figtree.yml", true, true},
	}

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsLoadConfigD2(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "dupval"})
	arr1 = append(arr1, StringOption{"../figtree.yml", true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{"../figtree.yml", true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{"figtree.yml", true, "d2str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": StringOption{"../figtree.yml", true, "d1map1val0"},
			"key1": StringOption{"figtree.yml", true, "d2map1val1"},
			"key2": StringOption{"figtree.yml", true, "d2map1val2"},
			"dup":  StringOption{"figtree.yml", true, "d2dupval"},
		},
		Int1:   IntOption{"figtree.yml", true, 222},
		Float1: Float32Option{"figtree.yml", true, 2.22},
		Bool1:  BoolOption{"figtree.yml", true, false},
	}

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsLoadConfigD1(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1")
	defer os.Chdir("..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d1arr1val2"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "dupval"})

	expected := TestOptions{
		String1:    StringOption{"figtree.yml", true, "d1str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": StringOption{"figtree.yml", true, "d1map1val0"},
			"key1": StringOption{"figtree.yml", true, "d1map1val1"},
			"dup":  StringOption{"figtree.yml", true, "d1dupval"},
		},
		Int1:   IntOption{"figtree.yml", true, 111},
		Float1: Float32Option{"figtree.yml", true, 1.11},
		Bool1:  BoolOption{"figtree.yml", true, true},
	}

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsCorrupt(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1")
	defer os.Chdir("..")

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("corrupt.yml", &opts)
	assert.NotNil(t, err)
}

func TestBuiltinLoadConfigD3(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")

	arr1 := []string{}
	arr1 = append(arr1, "d3arr1val1")
	arr1 = append(arr1, "d3arr1val2")
	arr1 = append(arr1, "dupval")
	arr1 = append(arr1, "d2arr1val1")
	arr1 = append(arr1, "d2arr1val2")
	arr1 = append(arr1, "d1arr1val1")
	arr1 = append(arr1, "d1arr1val2")

	expected := TestBuiltin{
		String1:    "d3str1val1",
		LeaveEmpty: "",
		Array1:     arr1,
		Map1: map[string]string{
			"key0": "d1map1val0",
			"key1": "d2map1val1",
			"key2": "d3map1val2",
			"key3": "d3map1val3",
			"dup":  "d3dupval",
		},
		Int1:   333,
		Float1: 3.33,
		Bool1:  true,
	}

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinLoadConfigD2(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	arr1 := []string{}
	arr1 = append(arr1, "d2arr1val1")
	arr1 = append(arr1, "d2arr1val2")
	arr1 = append(arr1, "dupval")
	arr1 = append(arr1, "d1arr1val1")
	arr1 = append(arr1, "d1arr1val2")

	expected := TestBuiltin{
		String1:    "d2str1val1",
		LeaveEmpty: "",
		Array1:     arr1,
		Map1: map[string]string{
			"key0": "d1map1val0",
			"key1": "d2map1val1",
			"key2": "d2map1val2",
			"dup":  "d2dupval",
		},
		Int1:   222,
		Float1: 2.22,
		// note this will be true from d1/figtree.yml since the
		// d1/d2/figtree.yml set it to false which is a zero value
		Bool1: true,
	}

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinLoadConfigD1(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1")
	defer os.Chdir("..")

	arr1 := []string{}
	arr1 = append(arr1, "d1arr1val1")
	arr1 = append(arr1, "d1arr1val2")
	arr1 = append(arr1, "dupval")

	expected := TestBuiltin{
		String1:    "d1str1val1",
		LeaveEmpty: "",
		Array1:     arr1,
		Map1: map[string]string{
			"key0": "d1map1val0",
			"key1": "d1map1val1",
			"dup":  "d1dupval",
		},
		Int1:   111,
		Float1: 1.11,
		Bool1:  true,
	}

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinCorrupt(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1")
	defer os.Chdir("..")

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("corrupt.yml", &opts)
	assert.NotNil(t, err)
}

func TestOptionsLoadConfigDefaults(t *testing.T) {
	opts := TestOptions{
		String1:    NewStringOption("defaultVal1"),
		LeaveEmpty: NewStringOption("emptyVal1"),
		Int1:       NewIntOption(999),
		Float1:     NewFloat32Option(9.99),
		Bool1:      NewBoolOption(true),
	}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{"figtree.yml", true, "dupval"})
	arr1 = append(arr1, StringOption{"../figtree.yml", true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{"../figtree.yml", true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{"figtree.yml", true, "d2str1val1"},
		LeaveEmpty: StringOption{"default", true, "emptyVal1"},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": StringOption{"../figtree.yml", true, "d1map1val0"},
			"key1": StringOption{"figtree.yml", true, "d2map1val1"},
			"key2": StringOption{"figtree.yml", true, "d2map1val2"},
			"dup":  StringOption{"figtree.yml", true, "d2dupval"},
		},
		Int1:   IntOption{"figtree.yml", true, 222},
		Float1: Float32Option{"figtree.yml", true, 2.22},
		Bool1:  BoolOption{"figtree.yml", true, false},
	}

	fig := newFigTreeFromEnv()
	_, err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	require.Exactly(t, expected, opts)
}

func TestMergeMapsWithNull(t *testing.T) {
	dest := map[string]interface{}{
		"requires": map[string]interface{}{
			"pkgA": nil,
			"pkgB": ">1.2.3",
		},
	}

	src := map[string]interface{}{
		"requires": map[string]interface{}{
			"pkgC": "<1.2.3",
			"pkgD": nil,
		},
	}

	Merge(dest, src)

	expected := map[string]interface{}{
		"requires": map[string]interface{}{
			"pkgA": nil,
			"pkgB": ">1.2.3",
			"pkgC": "<1.2.3",
			"pkgD": nil,
		},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeMapsIntoStructWithNull(t *testing.T) {
	src1 := map[string]interface{}{
		"requires": map[string]interface{}{
			"pkgA": nil,
			"pkgB": ">1.2.3",
		},
	}

	src2 := map[string]interface{}{
		"requires": map[string]interface{}{
			"pkgC": "<1.2.3",
			"pkgD": nil,
		},
	}

	dest := MakeMergeStruct(src1, src2)
	Merge(dest, src1)
	Merge(dest, src2)

	expected := &struct {
		Requires struct {
			PkgA interface{} `json:"pkgA" yaml:"pkgA"`
			PkgB string      `json:"pkgB" yaml:"pkgB"`
			PkgC string      `json:"pkgC" yaml:"pkgC"`
			PkgD interface{} `json:"pkgD" yaml:"pkgD"`
		} `json:"requires" yaml:"requires"`
	}{
		struct {
			PkgA interface{} `json:"pkgA" yaml:"pkgA"`
			PkgB string      `json:"pkgB" yaml:"pkgB"`
			PkgC string      `json:"pkgC" yaml:"pkgC"`
			PkgD interface{} `json:"pkgD" yaml:"pkgD"`
		}{
			PkgA: nil,
			PkgB: ">1.2.3",
			PkgC: "<1.2.3",
			PkgD: nil,
		},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeStringIntoStringOption(t *testing.T) {
	src1 := struct {
		Value StringOption
	}{}

	src2 := struct {
		Value string
	}{"val1"}

	dest := MakeMergeStruct(src1, src2)

	Merge(dest, src1)
	Merge(dest, src2)

	expected := &struct {
		Value StringOption
	}{StringOption{"merge", true, "val1"}}
	assert.Equal(t, expected, dest)
}

func TestMergeStringOptions(t *testing.T) {
	src1 := struct {
		Value StringOption
	}{}

	src2 := struct {
		Value StringOption
	}{NewStringOption("val1")}

	dest := MakeMergeStruct(src1, src2)

	Merge(dest, src1)
	Merge(dest, src2)

	expected := &struct {
		Value StringOption
	}{StringOption{"default", true, "val1"}}
	assert.Equal(t, expected, dest)
}

func TestMergeMapStringIntoStringOption(t *testing.T) {
	src1 := map[string]interface{}{
		"map": MapStringOption{},
	}

	src2 := map[string]interface{}{
		"map": MapStringOption{
			"key": NewStringOption("val1"),
		},
	}
	dest := MakeMergeStruct(src1, src2)

	Merge(dest, src1)
	Merge(dest, src2)

	expected := &struct {
		Map struct {
			Key StringOption `json:"key" yaml:"key"`
		} `json:"map" yaml:"map"`
	}{
		Map: struct {
			Key StringOption `json:"key" yaml:"key"`
		}{StringOption{"default", true, "val1"}},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeMapStringOptions(t *testing.T) {
	src1 := struct {
		Value StringOption
	}{}

	src2 := struct {
		Value StringOption
	}{NewStringOption("val1")}

	dest := MakeMergeStruct(src1, src2)

	Merge(dest, src1)
	Merge(dest, src2)

	expected := &struct {
		Value StringOption
	}{StringOption{"default", true, "val1"}}
	assert.Equal(t, expected, dest)
}

func TestMergeMapWithStruct(t *testing.T) {
	dest := map[string]interface{}{
		"mapkey": "mapval1",
		"map": map[string]interface{}{
			"mapkey":  "mapval2",
			"nullkey": nil,
		},
	}

	src := struct {
		StructField string
		Map         struct {
			StructField string
		}
	}{
		StructField: "field1",
		Map: struct {
			StructField string
		}{
			StructField: "field2",
		},
	}

	m := NewMerger()
	m.mergeStructs(reflect.ValueOf(&dest), reflect.ValueOf(&src))

	expected := map[string]interface{}{
		"mapkey":       "mapval1",
		"struct-field": "field1",
		"map": map[string]interface{}{
			"mapkey":       "mapval2",
			"nullkey":      nil,
			"struct-field": "field2",
		},
	}
	assert.Equal(t, expected, dest)

}

func TestMergeStructWithMap(t *testing.T) {
	dest := struct {
		StructField string
		Mapkey      string
		Map         struct {
			StructField string
			Mapkey      string
		}
	}{
		StructField: "field1",
		Map: struct {
			StructField string
			Mapkey      string
		}{
			StructField: "field2",
		},
	}

	src := map[string]interface{}{
		"mapkey": "mapval1",
		"map": map[string]interface{}{
			"mapkey":  "mapval2",
			"nullkey": nil,
		},
	}

	merged := MakeMergeStruct(&dest, &src)
	Merge(merged, &dest)
	Merge(merged, &src)

	expected := struct {
		Map struct {
			Mapkey      string
			Nullkey     interface{} `json:"nullkey" yaml:"nullkey"`
			StructField string
		}
		Mapkey      string
		StructField string
	}{
		Map: struct {
			Mapkey      string
			Nullkey     interface{} `json:"nullkey" yaml:"nullkey"`
			StructField string
		}{
			Mapkey:      "mapval2",
			StructField: "field2",
		},
		Mapkey:      "mapval1",
		StructField: "field1",
	}
	assert.Equal(t, &expected, merged)
}

func TestMergeStructUsingOptionsWithMap(t *testing.T) {
	dest := struct {
		Bool    BoolOption
		Byte    ByteOption
		Float32 Float32Option
		Float64 Float64Option
		Int16   Int16Option
		Int32   Int32Option
		Int64   Int64Option
		Int8    Int8Option
		Int     IntOption
		Rune    RuneOption
		String  StringOption
		Uint16  Uint16Option
		Uint32  Uint32Option
		Uint64  Uint64Option
		Uint8   Uint8Option
		Uint    UintOption
	}{}

	src := map[string]interface{}{
		"bool":    true,
		"byte":    byte(10),
		"float32": float32(1.23),
		"float64": float64(2.34),
		"int16":   int16(123),
		"int32":   int32(234),
		"int64":   int64(345),
		"int8":    int8(127),
		"int":     int(456),
		"rune":    rune('a'),
		"string":  "stringval",
		"uint16":  uint16(123),
		"uint32":  uint32(234),
		"uint64":  uint64(345),
		"uint8":   uint8(255),
		"uint":    uint(456),
	}

	Merge(&dest, &src)

	expected := struct {
		Bool    BoolOption
		Byte    ByteOption
		Float32 Float32Option
		Float64 Float64Option
		Int16   Int16Option
		Int32   Int32Option
		Int64   Int64Option
		Int8    Int8Option
		Int     IntOption
		Rune    RuneOption
		String  StringOption
		Uint16  Uint16Option
		Uint32  Uint32Option
		Uint64  Uint64Option
		Uint8   Uint8Option
		Uint    UintOption
	}{
		Bool:    BoolOption{"merge", true, true},
		Byte:    ByteOption{"merge", true, byte(10)},
		Float32: Float32Option{"merge", true, float32(1.23)},
		Float64: Float64Option{"merge", true, float64(2.34)},
		Int16:   Int16Option{"merge", true, int16(123)},
		Int32:   Int32Option{"merge", true, int32(234)},
		Int64:   Int64Option{"merge", true, int64(345)},
		Int8:    Int8Option{"merge", true, int8(127)},
		Int:     IntOption{"merge", true, int(456)},
		Rune:    RuneOption{"merge", true, rune('a')},
		String:  StringOption{"merge", true, "stringval"},
		Uint16:  Uint16Option{"merge", true, uint16(123)},
		Uint32:  Uint32Option{"merge", true, uint32(234)},
		Uint64:  Uint64Option{"merge", true, uint64(345)},
		Uint8:   Uint8Option{"merge", true, uint8(255)},
		Uint:    UintOption{"merge", true, uint(456)},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeMapWithStructUsingOptions(t *testing.T) {
	dest := map[string]interface{}{
		"bool":    false,
		"byte":    byte(0),
		"float32": float32(0),
		"float64": float64(0),
		"int16":   int16(0),
		"int32":   int32(0),
		"int64":   int64(0),
		"int8":    int8(0),
		"int":     int(0),
		"rune":    rune(0),
		"string":  "",
		"uint16":  uint16(0),
		"uint32":  uint32(0),
		"uint64":  uint64(0),
		"uint8":   uint8(0),
		"uint":    uint(0),
	}

	src := struct {
		Bool    BoolOption
		Byte    ByteOption
		Float32 Float32Option `yaml:"float32"`
		Float64 Float64Option `yaml:"float64"`
		Int16   Int16Option   `yaml:"int16"`
		Int32   Int32Option   `yaml:"int32"`
		Int64   Int64Option   `yaml:"int64"`
		Int8    Int8Option    `yaml:"int8"`
		Int     IntOption
		Rune    RuneOption
		String  StringOption
		Uint16  Uint16Option `yaml:"uint16"`
		Uint32  Uint32Option `yaml:"uint32"`
		Uint64  Uint64Option `yaml:"uint64"`
		Uint8   Uint8Option  `yaml:"uint8"`
		Uint    UintOption
	}{
		Bool:    NewBoolOption(true),
		Byte:    NewByteOption(10),
		Float32: NewFloat32Option(1.23),
		Float64: NewFloat64Option(2.34),
		Int16:   NewInt16Option(123),
		Int32:   NewInt32Option(234),
		Int64:   NewInt64Option(345),
		Int8:    NewInt8Option(127),
		Int:     NewIntOption(456),
		Rune:    NewRuneOption('a'),
		String:  NewStringOption("stringval"),
		Uint16:  NewUint16Option(123),
		Uint32:  NewUint32Option(234),
		Uint64:  NewUint64Option(345),
		Uint8:   NewUint8Option(255),
		Uint:    NewUintOption(456),
	}

	Merge(&dest, &src)
	expected := map[string]interface{}{
		"bool":    true,
		"byte":    byte(10),
		"float32": float32(1.23),
		"float64": float64(2.34),
		"int16":   int16(123),
		"int32":   int32(234),
		"int64":   int64(345),
		"int8":    int8(127),
		"int":     int(456),
		"rune":    rune('a'),
		"string":  "stringval",
		"uint16":  uint16(123),
		"uint32":  uint32(234),
		"uint64":  uint64(345),
		"uint8":   uint8(255),
		"uint":    uint(456),
	}
	assert.Equal(t, expected, dest)
}

func TestMergeStructUsingListOptionsWithMap(t *testing.T) {
	dest := struct {
		Strings ListStringOption
	}{
		Strings: ListStringOption{
			NewStringOption("abc"),
		},
	}

	src := map[string]interface{}{
		"strings": []string{
			"abc",
			"def",
		},
	}

	Merge(&dest, &src)

	expected := struct {
		Strings ListStringOption
	}{
		ListStringOption{
			StringOption{"default", true, "abc"},
			StringOption{"merge", true, "def"},
		},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeMapWithStructUsingListOptions(t *testing.T) {
	dest := map[string]interface{}{
		"strings": []string{"abc"},
	}

	src := struct {
		Strings ListStringOption
	}{
		Strings: ListStringOption{
			NewStringOption("abc"),
			NewStringOption("def"),
		},
	}

	Merge(&dest, &src)
	expected := map[string]interface{}{
		"strings": []string{"abc", "def"},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeStructUsingMapOptionsWithMap(t *testing.T) {
	dest := struct {
		Strings MapStringOption
	}{}

	src := map[string]interface{}{
		"strings": map[string]interface{}{
			"key1": "val1",
			"key2": "val2",
		},
	}

	Merge(&dest, &src)

	expected := struct {
		Strings MapStringOption
	}{
		Strings: MapStringOption{
			"key1": StringOption{"merge", true, "val1"},
			"key2": StringOption{"merge", true, "val2"},
		},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeMapWithStructUsingMapOptions(t *testing.T) {
	dest := map[string]interface{}{
		"strings": map[string]string{},
	}

	src := struct {
		Strings MapStringOption
	}{
		Strings: MapStringOption{
			"key1": NewStringOption("val1"),
			"key2": NewStringOption("val2"),
		},
	}

	Merge(&dest, &src)
	expected := map[string]interface{}{
		"strings": map[string]string{
			"key1": "val1",
			"key2": "val2",
		},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeStructsWithSrcEmbedded(t *testing.T) {
	dest := struct {
		FieldName string
	}{}

	type embedded struct {
		FieldName string
	}

	src := struct {
		embedded
	}{
		embedded: embedded{
			FieldName: "field1",
		},
	}

	m := NewMerger()
	m.mergeStructs(reflect.ValueOf(&dest), reflect.ValueOf(&src))

	expected := struct {
		FieldName string
	}{
		FieldName: "field1",
	}
	assert.Equal(t, expected, dest)
}

func TestMergeStructsWithDestEmbedded(t *testing.T) {
	type embedded struct {
		FieldName string
	}

	dest := struct {
		embedded
	}{}

	src := struct {
		FieldName string
	}{
		FieldName: "field1",
	}

	m := NewMerger()
	m.mergeStructs(reflect.ValueOf(&dest), reflect.ValueOf(&src))

	expected := struct {
		embedded
	}{
		embedded: embedded{
			FieldName: "field1",
		},
	}
	assert.Equal(t, expected, dest)
}

func TestMakeMergeStruct(t *testing.T) {
	input := map[string]interface{}{
		"mapkey": "mapval1",
		"map": map[string]interface{}{
			"mapkey": "mapval2",
		},
		"nilmap": nil,
	}

	got := MakeMergeStruct(input)

	Merge(got, &input)

	assert.Equal(t, input["mapkey"], reflect.ValueOf(got).Elem().FieldByName("Mapkey").Interface())
	assert.Equal(t, struct {
		Mapkey string `json:"mapkey" yaml:"mapkey"`
	}{"mapval2"}, reflect.ValueOf(got).Elem().FieldByName("Map").Interface())
	assert.Equal(t, input["map"].(map[string]interface{})["mapkey"], reflect.ValueOf(got).Elem().FieldByName("Map").FieldByName("Mapkey").Interface())
}

func TestMakeMergeStructWithDups(t *testing.T) {
	input := map[string]interface{}{
		"mapkey": "mapval1",
	}

	s := struct {
		Mapkey string
	}{
		Mapkey: "mapval2",
	}

	got := MakeMergeStruct(input, s)
	Merge(got, &input)
	assert.Equal(t, &struct {
		Mapkey string `json:"mapkey" yaml:"mapkey"`
	}{"mapval1"}, got)

	got = MakeMergeStruct(s, input)
	Merge(got, &s)
	assert.Equal(t, &struct{ Mapkey string }{"mapval2"}, got)
}

func TestMakeMergeStructWithInline(t *testing.T) {
	type Inner struct {
		InnerString string
	}

	outer := struct {
		Inner       `figtree:",inline"`
		OuterString string
	}{}

	other := struct {
		InnerString string
		OtherString string
	}{}

	got := MakeMergeStruct(outer, other)
	assert.IsType(t, (*struct {
		InnerString string
		OtherString string
		OuterString string
	})(nil), got)

	otherMap := map[string]interface{}{
		"inner-string": "inner",
		"other-string": "other",
	}

	got = MakeMergeStruct(outer, otherMap)
	assert.IsType(t, (*struct {
		InnerString string `json:"inner-string" yaml:"inner-string"`
		OtherString string `json:"other-string" yaml:"other-string"`
		OuterString string
	})(nil), got)
}

func TestMakeMergeStructWithYaml(t *testing.T) {
	input := "foo-bar: foo-val\n"
	data := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(input), &data)
	assert.NoError(t, err)

	// turn map data into a struct
	got := MakeMergeStruct(data)
	// then assign the data back into that struct
	Merge(got, data)

	expected := &struct {
		FooBar string `json:"foo-bar" yaml:"foo-bar"`
	}{
		"foo-val",
	}
	assert.Equal(t, expected, got)

	// make sure the new structure serializes back to the original document
	output, err := yaml.Marshal(expected)
	assert.NoError(t, err)
	assert.Equal(t, input, string(output))
}

func TestMakeMergeStructWithJson(t *testing.T) {
	input := `{"foo-bar":"foo-val"}`
	data := map[string]interface{}{}
	err := json.Unmarshal([]byte(input), &data)
	assert.NoError(t, err)

	// turn map data into a struct
	got := MakeMergeStruct(data)
	// then assign the data back into that struct
	Merge(got, data)

	expected := &struct {
		FooBar string `json:"foo-bar" yaml:"foo-bar"`
	}{
		"foo-val",
	}
	assert.Equal(t, expected, got)

	// make sure the new structure serializes back to the original document
	output, err := json.Marshal(expected)
	assert.NoError(t, err)
	assert.Equal(t, input, string(output))
}

func TestMergeWithZeros(t *testing.T) {
	var zero interface{}
	tests := []struct {
		info info
		dest map[string]interface{}
		src  map[string]interface{}
		want map[string]interface{}
	}{
		{
			info: info{"zero to nil", line()},
			dest: map[string]interface{}{},
			src: map[string]interface{}{
				"value": zero,
			},
			want: map[string]interface{}{
				"value": zero,
			},
		},
		{
			info: info{"zero to zero", line()},
			dest: map[string]interface{}{
				"value": zero,
			},
			src: map[string]interface{}{
				"value": zero,
			},
			want: map[string]interface{}{
				"value": zero,
			},
		},
		{
			info: info{"zero to StringOption", line()},
			dest: map[string]interface{}{
				"value": StringOption{},
			},
			src: map[string]interface{}{
				"value": zero,
			},
			want: map[string]interface{}{
				"value": StringOption{},
			},
		},
		{
			info: info{"StringOption to zero", line()},
			dest: map[string]interface{}{
				"value": zero,
			},
			src: map[string]interface{}{
				"value": StringOption{},
			},
			want: map[string]interface{}{
				"value": StringOption{},
			},
		},
		{
			info: info{"list zero to nil", line()},
			dest: map[string]interface{}{
				"value": nil,
			},
			src: map[string]interface{}{
				"value": []interface{}{zero},
			},
			want: map[string]interface{}{
				"value": []interface{}{zero},
			},
		},
		{
			info: info{"list zero to empty", line()},
			dest: map[string]interface{}{
				"value": []interface{}{},
			},
			src: map[string]interface{}{
				"value": []interface{}{zero},
			},
			want: map[string]interface{}{
				"value": []interface{}{zero},
			},
		},
		{
			info: info{"list zero to StringOption", line()},
			dest: map[string]interface{}{
				"value": []interface{}{StringOption{}},
			},
			src: map[string]interface{}{
				"value": []interface{}{zero},
			},
			want: map[string]interface{}{
				"value": []interface{}{StringOption{}, zero},
			},
		},
		{
			info: info{"list StringOption to zero", line()},
			dest: map[string]interface{}{
				"value": []interface{}{zero},
			},
			src: map[string]interface{}{
				"value": []interface{}{StringOption{}},
			},
			want: map[string]interface{}{
				"value": []interface{}{zero, StringOption{}},
			},
		},
		{
			info: info{"list StringOption to empty", line()},
			dest: map[string]interface{}{
				"value": []interface{}{},
			},
			src: map[string]interface{}{
				"value": []interface{}{StringOption{}},
			},
			want: map[string]interface{}{
				"value": []interface{}{StringOption{}},
			},
		},
		{
			info: info{"zero to ListStringOption", line()},
			dest: map[string]interface{}{
				"value": ListStringOption{StringOption{}},
			},
			src: map[string]interface{}{
				"value": []interface{}{zero},
			},
			want: map[string]interface{}{
				"value": ListStringOption{StringOption{}},
			},
		},
		{
			info: info{"ListStringOption to zero", line()},
			dest: map[string]interface{}{
				"value": []interface{}{zero},
			},
			src: map[string]interface{}{
				"value": ListStringOption{StringOption{}},
			},
			want: map[string]interface{}{
				"value": []interface{}{zero},
			},
		},
		{
			info: info{"map zero to nil", line()},
			dest: map[string]interface{}{
				"value": nil,
			},
			src: map[string]interface{}{
				"value": map[string]interface{}{
					"key": zero,
				},
			},
			want: map[string]interface{}{
				"value": map[string]interface{}{
					"key": zero,
				},
			},
		},
		{
			info: info{"map zero to empty", line()},
			dest: map[string]interface{}{
				"value": map[string]interface{}{},
			},
			src: map[string]interface{}{
				"value": map[string]interface{}{
					"key": zero,
				},
			},
			want: map[string]interface{}{
				"value": map[string]interface{}{
					"key": zero,
				},
			},
		},
		{
			info: info{"MapStringOption to zero", line()},
			dest: map[string]interface{}{
				"value": zero,
			},
			src: map[string]interface{}{
				"value": MapStringOption{
					"key": StringOption{},
				},
			},
			want: map[string]interface{}{
				"value": MapStringOption{
					"key": StringOption{},
				},
			},
		},
		{
			info: info{"map zero to StringOption", line()},
			dest: map[string]interface{}{
				"value": MapStringOption{
					"key": StringOption{},
				},
			},
			src: map[string]interface{}{
				"value": zero,
			},
			want: map[string]interface{}{
				"value": MapStringOption{
					"key": StringOption{},
				},
			},
		},
		{
			info: info{"map zero key to StringOption", line()},
			dest: map[string]interface{}{
				"value": MapStringOption{
					"key": StringOption{},
				},
			},
			src: map[string]interface{}{
				"value": map[string]interface{}{
					"key": zero,
				},
			},
			want: map[string]interface{}{
				"value": MapStringOption{
					"key": StringOption{},
				},
			},
		},
		{
			info: info{"map StringOption to zero key", line()},
			dest: map[string]interface{}{
				"value": map[string]interface{}{
					"key": zero,
				},
			},
			src: map[string]interface{}{
				"value": MapStringOption{
					"key": StringOption{},
				},
			},
			want: map[string]interface{}{
				"value": map[string]interface{}{
					"key": StringOption{},
				},
			},
		},
	}

	for _, tt := range tests {
		require.True(t,
			t.Run(tt.info.name, func(t *testing.T) {
				// assert.NotPanics(t, func() {
				Log.Debugf("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
				Log.Debugf("%s", tt.info.name)
				Log.Debugf("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
				Merge(&tt.dest, &tt.src)
				// })
				assert.Equal(t, tt.want, tt.dest, tt.info.line)

				got := MakeMergeStruct(tt.dest)
				Merge(got, tt.dest)
				Merge(got, tt.src)

				expected := MakeMergeStruct(tt.want)
				Merge(expected, tt.want)

				assert.Equal(t, expected, got, tt.info.line)
			}),
		)
	}
}

func TestMergeStructsWithZeros(t *testing.T) {
	var zero interface{}
	tests := []struct {
		info info
		dest interface{}
		src  interface{}
		want interface{}
		line string
	}{
		{
			info: info{"bare nil", line()},
			dest: struct {
				Value interface{}
			}{},
			src: struct {
				Value interface{}
			}{zero},
			want: struct {
				Value interface{}
			}{zero},
		},
		{
			info: info{"bare zero", line()},
			dest: struct {
				Value interface{}
			}{zero},
			src: struct {
				Value interface{}
			}{zero},
			want: struct {
				Value interface{}
			}{zero},
		},
		{
			info: info{"bare StringOption", line()},
			dest: struct {
				Value interface{}
			}{StringOption{}},
			src: struct {
				Value interface{}
			}{StringOption{}},
			want: struct {
				Value interface{}
			}{StringOption{}},
		},
		{
			info: info{"bare StringOptions to zero", line()},
			dest: struct {
				Value interface{}
			}{zero},
			src: struct {
				Value StringOption
			}{StringOption{}},
			want: struct {
				Value interface{}
			}{zero},
		},
		{
			info: info{"list zero to nil", line()},
			dest: struct {
				Value interface{}
			}{},
			src: struct {
				Value interface{}
			}{[]interface{}{zero}},
			want: struct {
				Value interface{}
			}{[]interface{}{zero}},
		},
		{
			info: info{"list zero to empty", line()},
			dest: struct {
				Value interface{}
			}{[]interface{}{}},
			src: struct {
				Value interface{}
			}{[]interface{}{zero}},
			want: struct {
				Value interface{}
			}{[]interface{}{zero}},
		},
		{
			info: info{"list zero to StringOption", line()},
			dest: struct {
				Value interface{}
			}{[]interface{}{StringOption{}}},
			src: struct {
				Value interface{}
			}{[]interface{}{zero}},
			want: struct {
				Value interface{}
			}{[]interface{}{StringOption{}, zero}},
		},
		{
			info: info{"list StringOption to zero", line()},
			dest: struct {
				Value interface{}
			}{[]interface{}{zero}},
			src: struct {
				Value interface{}
			}{[]interface{}{StringOption{}}},
			want: struct {
				Value interface{}
			}{[]interface{}{zero, StringOption{}}},
		},
		{
			info: info{"list ListStringOption to empty list", line()},
			line: line(),
			dest: struct {
				Value interface{}
			}{[]interface{}{}},
			src: struct {
				Value ListStringOption
			}{ListStringOption{StringOption{}}},
			want: struct {
				Value interface{}
			}{[]interface{}{zero}},
		},
		{

			info: info{"list zero list to ListStringOption", line()},
			dest: struct {
				Value ListStringOption
			}{ListStringOption{StringOption{}}},
			src: struct {
				Value interface{}
			}{[]interface{}{zero}},
			want: struct {
				Value ListStringOption
			}{ListStringOption{StringOption{}}},
		},
		{
			info: info{"list ListStringOption to zero list", line()},
			dest: struct {
				Value interface{}
			}{[]interface{}{zero}},
			src: struct {
				Value ListStringOption
			}{ListStringOption{StringOption{}}},
			want: struct {
				Value interface{}
			}{[]interface{}{zero}},
		},
		{
			info: info{"map zero to nil", line()},
			dest: struct {
				Value interface{}
			}{},
			src: struct {
				Value interface{}
			}{map[string]interface{}{"key": zero}},
			want: struct {
				Value interface{}
			}{map[string]interface{}{"key": zero}},
		},
		{
			info: info{"map zero to empty", line()},
			dest: struct {
				Value interface{}
			}{map[string]interface{}{}},
			src: struct {
				Value interface{}
			}{map[string]interface{}{"key": zero}},
			want: struct {
				Value interface{}
			}{map[string]interface{}{"key": zero}},
		},
		{
			info: info{"map StringOption to zero", line()},
			dest: struct {
				Value interface{}
			}{zero},
			src: struct {
				Value interface{}
			}{map[string]interface{}{"key": zero}},
			want: struct {
				Value interface{}
			}{map[string]interface{}{"key": zero}},
		},
		{
			info: info{"MapStringOption StringOption to zero", line()},
			dest: struct {
				Value interface{}
			}{zero},
			src: struct {
				Value interface{}
			}{MapStringOption{
				"key": StringOption{},
			}},
			want: struct {
				Value interface{}
			}{MapStringOption{
				"key": StringOption{},
			}},
		},
		{
			info: info{"zero to MapStringOption StringOption", line()},
			dest: struct {
				Value interface{}
			}{MapStringOption{
				"key": StringOption{},
			}},
			src: struct {
				Value interface{}
			}{zero},
			want: struct {
				Value interface{}
			}{MapStringOption{
				"key": StringOption{},
			}},
		},
		{
			info: info{"map zero to MapStringOption StringOption", line()},
			dest: struct {
				Value interface{}
			}{MapStringOption{
				"key": StringOption{},
			}},
			src: struct {
				Value interface{}
			}{map[string]interface{}{
				"key": zero,
			}},
			want: struct {
				Value interface{}
			}{MapStringOption{
				"key": StringOption{},
			}},
		},
	}
	for _, tt := range tests {
		require.True(t,
			t.Run(tt.info.name, func(t *testing.T) {
				// assert.NotPanics(t, func() {
				Log.Debugf("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
				Log.Debugf("%s", tt.info.name)
				Log.Debugf("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
				Merge(&tt.dest, &tt.src)
				// })
				assert.Equal(t, tt.want, tt.dest, tt.info.line)

				got := MakeMergeStruct(tt.dest)
				Merge(got, tt.dest)
				Merge(got, tt.src)

				expected := MakeMergeStruct(tt.want)
				Merge(expected, tt.want)

				assert.Equal(t, expected, got, tt.info.line)

			}),
		)
	}
}

func TestMergeStructsWithPreservedMaps(t *testing.T) {
	tests := []struct {
		info   info
		src    interface{}
		want   interface{}
		merger *Merger
	}{
		{
			info: info{"convert map to struct by default", line()},
			src: map[string]interface{}{
				"map": map[string]string{"key": "value"},
			},
			want: &struct {
				Map struct {
					Key string `json:"key" yaml:"key"`
				} `json:"map" yaml:"map"`
			}{},
			merger: NewMerger(),
		}, {
			info: info{"preserve map when converting to struct", line()},
			src: map[string]interface{}{
				"map":   map[string]string{"key": "value"},
				"other": map[string]string{"key": "value"},
			},
			want: &struct {
				Map   map[string]string `json:"map" yaml:"map"`
				Other struct {
					Key string `json:"key" yaml:"key"`
				} `json:"other" yaml:"other"`
			}{},
			merger: NewMerger(PreserveMap("map")),
		},
	}

	for _, tt := range tests {
		require.True(t,
			t.Run(tt.info.name, func(t *testing.T) {
				got := tt.merger.MakeMergeStruct(tt.src)
				assert.Equal(t, tt.want, got)
			}),
		)
	}
}

func TestFigtreePreProcessor(t *testing.T) {
	input := []byte(`
bad-name: good-value
good-name: bad-value
ok-name: want-array
`)

	pp := func(in []byte) ([]byte, error) {
		raw := map[string]interface{}{}
		if err := yaml.Unmarshal(in, &raw); err != nil {
			return in, err
		}

		// rename "bad-name" key to "fixed-name"
		if val, ok := raw["bad-name"]; ok {
			delete(raw, "bad-name")
			raw["fixed-name"] = val
		}

		// reset "bad-value" key to "good-value"
		if val, ok := raw["good-name"]; ok {
			if t, ok := val.(string); ok && t != "fixed-value" {
				raw["good-name"] = "fixed-value"
			}
		}

		// migrate "ok-name" value from string to list of strings
		if val, ok := raw["ok-name"]; ok {
			if t, ok := val.(string); ok {
				raw["ok-name"] = []string{t}
			}
		}
		return yaml.Marshal(raw)
	}

	fig := newFigTreeFromEnv(WithPreProcessor(pp))

	dest := struct {
		FixedName string   `yaml:"fixed-name"`
		GoodName  string   `yaml:"good-name"`
		OkName    []string `yaml:"ok-name"`
	}{}

	want := struct {
		FixedName string   `yaml:"fixed-name"`
		GoodName  string   `yaml:"good-name"`
		OkName    []string `yaml:"ok-name"`
	}{"good-value", "fixed-value", []string{"want-array"}}

	_, err := fig.LoadConfigBytes(input, "test", &dest)
	assert.NoError(t, err)
	assert.Equal(t, want, dest)
}
