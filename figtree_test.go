package figtree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"emperror.dev/errors"
	logging "gopkg.in/op/go-logging.v1"
	yaml "gopkg.in/yaml.v3"

	"github.com/coryb/walky"
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

func newFigTreeFromEnv(opts ...CreateOption) *FigTree {
	cwd, _ := os.Getwd()
	opts = append([]CreateOption{
		WithHome(os.Getenv("HOME")),
		WithCwd(cwd),
		WithEnvPrefix("FIGTREE"),
	}, opts...)

	return NewFigTree(opts...)
}

func tSrc(s string, l, c int) SourceLocation {
	return NewSource(s, WithLocation(&FileCoordinate{Line: l, Column: c}))
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
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 3, 5), true, "d3arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 4, 5), true, "d3arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 5, 5), true, "dupval"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 3, 5), true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 4, 5), true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("../../figtree.yml", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../../figtree.yml", 4, 5), true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{tSrc("figtree.yml", 1, 7), true, "d3str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {tSrc("../../figtree.yml", 7, 9), true, "d1map1val0"},
			"key1": {tSrc("../figtree.yml", 7, 9), true, "d2map1val1"},
			"key2": {tSrc("figtree.yml", 7, 9), true, "d3map1val2"},
			"key3": {tSrc("figtree.yml", 8, 9), true, "d3map1val3"},
			"dup":  {tSrc("figtree.yml", 9, 9), true, "d3dupval"},
		},
		Int1:   IntOption{tSrc("figtree.yml", 10, 7), true, 333},
		Float1: Float32Option{tSrc("figtree.yml", 11, 9), true, 3.33},
		Bool1:  BoolOption{tSrc("figtree.yml", 12, 8), true, true},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsLoadConfigD2(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1/d2"))
	t.Cleanup(func() {
		_ = os.Chdir("../..")
	})

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 3, 5), true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 4, 5), true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 5, 5), true, "dupval"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 4, 5), true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{tSrc("figtree.yml", 1, 7), true, "d2str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {tSrc("../figtree.yml", 7, 9), true, "d1map1val0"},
			"key1": {tSrc("figtree.yml", 7, 9), true, "d2map1val1"},
			"key2": {tSrc("figtree.yml", 8, 9), true, "d2map1val2"},
			"dup":  {tSrc("figtree.yml", 9, 9), true, "d2dupval"},
		},
		Int1:   IntOption{tSrc("figtree.yml", 10, 7), true, 222},
		Float1: Float32Option{tSrc("figtree.yml", 11, 9), true, 2.22},
		Bool1:  BoolOption{tSrc("figtree.yml", 12, 8), true, false},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsLoadConfigD1(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 4, 5), true, "d1arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 5, 5), true, "dupval"})

	expected := TestOptions{
		String1:    StringOption{tSrc("figtree.yml", 1, 7), true, "d1str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {tSrc("figtree.yml", 7, 9), true, "d1map1val0"},
			"key1": {tSrc("figtree.yml", 8, 9), true, "d1map1val1"},
			"dup":  {tSrc("figtree.yml", 9, 9), true, "d1dupval"},
		},
		Int1:   IntOption{tSrc("figtree.yml", 10, 7), true, 111},
		Float1: Float32Option{tSrc("figtree.yml", 11, 9), true, 1.11},
		Bool1:  BoolOption{tSrc("figtree.yml", 12, 8), true, true},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsCorrupt(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("corrupt.yml", &opts)
	assert.NotNil(t, err)
}

func TestBuiltinLoadConfigD3(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})

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
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinLoadConfigD2(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1/d2"))
	t.Cleanup(func() {
		_ = os.Chdir("../..")
	})

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
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinLoadConfigD1(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

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
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinCorrupt(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("corrupt.yml", &opts)
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
	require.NoError(t, os.Chdir("d1/d2"))
	t.Cleanup(func() {
		_ = os.Chdir("../..")
	})

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 3, 5), true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 4, 5), true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 5, 5), true, "dupval"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 4, 5), true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{tSrc("figtree.yml", 1, 7), true, "d2str1val1"},
		LeaveEmpty: StringOption{NewSource("default"), true, "emptyVal1"},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {tSrc("../figtree.yml", 7, 9), true, "d1map1val0"},
			"key1": {tSrc("figtree.yml", 7, 9), true, "d2map1val1"},
			"key2": {tSrc("figtree.yml", 8, 9), true, "d2map1val2"},
			"dup":  {tSrc("figtree.yml", 9, 9), true, "d2dupval"},
		},
		Int1:   IntOption{tSrc("figtree.yml", 10, 7), true, 222},
		Float1: Float32Option{tSrc("figtree.yml", 11, 9), true, 2.22},
		Bool1:  BoolOption{tSrc("figtree.yml", 12, 8), true, false},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)
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

	err := Merge(dest, src)
	require.NoError(t, err)

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
	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

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

	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

	expected := &struct {
		Value StringOption
	}{StringOption{NewSource("merge"), true, "val1"}}
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

	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

	expected := &struct {
		Value StringOption
	}{StringOption{NewSource("default"), true, "val1"}}
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

	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

	expected := &struct {
		Map struct {
			Key StringOption `json:"key" yaml:"key"`
		} `json:"map" yaml:"map"`
	}{
		Map: struct {
			Key StringOption `json:"key" yaml:"key"`
		}{StringOption{NewSource("default"), true, "val1"}},
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

	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

	expected := &struct {
		Value StringOption
	}{StringOption{NewSource("default"), true, "val1"}}
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
	changed, err := m.mergeStructs(reflect.ValueOf(&dest), newMergeSource(reflect.ValueOf(&src)), false)
	require.NoError(t, err)
	require.True(t, changed)

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
	err := Merge(merged, &dest)
	require.NoError(t, err)
	err = Merge(merged, &src)
	require.NoError(t, err)

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

func TestMergeStructWithMapArbitraryNaming(t *testing.T) {
	// Go struct field names should not matter
	dest := struct {
		MyStructField string `yaml:"struct-field"`
		MyMapkey      string `yaml:"mapkey"`
		MyMap         struct {
			MyStructField string `yaml:"struct-field"`
			MyMapkey      string `yaml:"mapkey"`
		} `yaml:"map"`
	}{
		MyStructField: "field1",
		MyMap: struct {
			MyStructField string `yaml:"struct-field"`
			MyMapkey      string `yaml:"mapkey"`
		}{
			MyStructField: "field2",
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
	err := Merge(merged, &dest)
	require.NoError(t, err)
	err = Merge(merged, &src)
	require.NoError(t, err)

	expected := struct {
		Map struct {
			Mapkey      string      `yaml:"mapkey"`
			Nullkey     interface{} `json:"nullkey" yaml:"nullkey"`
			StructField string      `yaml:"struct-field"`
		} `yaml:"map"`
		Mapkey      string `yaml:"mapkey"`
		StructField string `yaml:"struct-field"`
	}{
		Map: struct {
			Mapkey      string      `yaml:"mapkey"`
			Nullkey     interface{} `json:"nullkey" yaml:"nullkey"`
			StructField string      `yaml:"struct-field"`
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
		"bool":     true,
		"byte":     byte(10),
		"float-32": float32(1.23),
		"float-64": float64(2.34),
		"int-16":   int16(123),
		"int-32":   int32(234),
		"int-64":   int64(345),
		"int-8":    int8(127),
		"int":      int(456),
		"rune":     rune('a'),
		"string":   "stringval",
		"uint-16":  uint16(123),
		"uint-32":  uint32(234),
		"uint-64":  uint64(345),
		"uint-8":   uint8(255),
		"uint":     uint(456),
	}

	err := Merge(&dest, &src)
	require.NoError(t, err)

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
		Bool:    BoolOption{NewSource("merge"), true, true},
		Byte:    ByteOption{NewSource("merge"), true, byte(10)},
		Float32: Float32Option{NewSource("merge"), true, float32(1.23)},
		Float64: Float64Option{NewSource("merge"), true, float64(2.34)},
		Int16:   Int16Option{NewSource("merge"), true, int16(123)},
		Int32:   Int32Option{NewSource("merge"), true, int32(234)},
		Int64:   Int64Option{NewSource("merge"), true, int64(345)},
		Int8:    Int8Option{NewSource("merge"), true, int8(127)},
		Int:     IntOption{NewSource("merge"), true, int(456)},
		Rune:    RuneOption{NewSource("merge"), true, rune('a')},
		String:  StringOption{NewSource("merge"), true, "stringval"},
		Uint16:  Uint16Option{NewSource("merge"), true, uint16(123)},
		Uint32:  Uint32Option{NewSource("merge"), true, uint32(234)},
		Uint64:  Uint64Option{NewSource("merge"), true, uint64(345)},
		Uint8:   Uint8Option{NewSource("merge"), true, uint8(255)},
		Uint:    UintOption{NewSource("merge"), true, uint(456)},
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

	err := Merge(&dest, &src)
	require.NoError(t, err)

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

	err := Merge(&dest, &src)
	require.NoError(t, err)

	expected := struct {
		Strings ListStringOption
	}{
		ListStringOption{
			StringOption{NewSource("default"), true, "abc"},
			StringOption{NewSource("merge"), true, "def"},
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

	err := Merge(&dest, &src)
	require.NoError(t, err)

	expected := map[string]interface{}{
		"strings": []string{"abc", "def"},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeStructWithListUsingListOptions(t *testing.T) {
	dest := struct {
		Property []interface{}
	}{
		Property: []interface{}{
			"abc",
		},
	}

	src := struct {
		Property ListStringOption
	}{
		Property: ListStringOption{
			NewStringOption("abc"),
			NewStringOption("def"),
		},
	}

	err := Merge(&dest, &src)
	require.NoError(t, err)

	expected := struct {
		Property []interface{}
	}{
		Property: []interface{}{
			"abc",
			NewStringOption("def"),
		},
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

	err := Merge(&dest, &src)
	require.NoError(t, err)

	expected := struct {
		Strings MapStringOption
	}{
		Strings: MapStringOption{
			"key1": StringOption{NewSource("merge"), true, "val1"},
			"key2": StringOption{NewSource("merge"), true, "val2"},
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

	err := Merge(&dest, &src)
	require.NoError(t, err)

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
	changed, err := m.mergeStructs(reflect.ValueOf(&dest), newMergeSource(reflect.ValueOf(&src)), false)
	require.NoError(t, err)
	require.True(t, changed)

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
	changed, err := m.mergeStructs(reflect.ValueOf(&dest), newMergeSource(reflect.ValueOf(&src)), false)
	require.NoError(t, err)
	require.True(t, changed)

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

	err := Merge(got, &input)
	require.NoError(t, err)

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
	err := Merge(got, &input)
	require.NoError(t, err)

	assert.Equal(t, &struct {
		Mapkey string `json:"mapkey" yaml:"mapkey"`
	}{"mapval1"}, got)

	got = MakeMergeStruct(s, input)
	err = Merge(got, &s)
	require.NoError(t, err)

	assert.Equal(t, &struct{ Mapkey string }{"mapval2"}, got)
}

func TestMakeMergeStructWithInline(t *testing.T) {
	type Inner struct {
		InnerString StringOption `json:"inner-string" yaml:"inner-string"`
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
		InnerString StringOption `json:"inner-string" yaml:"inner-string"`
		OtherString string
		OuterString string
	})(nil), got)

	otherMap := map[string]interface{}{
		"inner-string": "inner",
		"other-string": "other",
	}

	got = MakeMergeStruct(outer, otherMap)
	assert.IsType(t, (*struct {
		InnerString StringOption `json:"inner-string" yaml:"inner-string"`
		OtherString string       `json:"other-string" yaml:"other-string"`
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
	err = Merge(got, data)
	require.NoError(t, err)

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
	err = Merge(got, data)
	require.NoError(t, err)

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
				"value": []interface{}{},
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
				"value": []interface{}{StringOption{}},
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
				"value": []interface{}{zero},
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
				"value": []interface{}{},
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
				err := Merge(&tt.dest, &tt.src)
				require.NoError(t, err)
				// })
				assert.Equal(t, tt.want, tt.dest, tt.info.line)

				got := MakeMergeStruct(tt.dest)
				err = Merge(got, tt.dest)
				require.NoError(t, err)
				err = Merge(got, tt.src)
				require.NoError(t, err)

				expected := MakeMergeStruct(tt.want)
				err = Merge(expected, tt.want)
				require.NoError(t, err)

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
			}{[]interface{}{}},
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
			}{[]interface{}{StringOption{}}},
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
			}{[]interface{}{zero}},
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
			}{[]interface{}{}},
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
				err := Merge(&tt.dest, &tt.src)
				require.NoError(t, err)
				// })
				assert.Equal(t, tt.want, tt.dest, tt.info.line)

				got := MakeMergeStruct(tt.dest)
				err = Merge(got, tt.dest)
				require.NoError(t, err)
				err = Merge(got, tt.src)
				require.NoError(t, err)

				expected := MakeMergeStruct(tt.want)
				err = Merge(expected, tt.want)
				require.NoError(t, err)

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
	var input yaml.Node
	err := yaml.Unmarshal([]byte(`
bad-name: good-value
good-name: bad-value
ok-name: want-array
`), &input)
	assert.NoError(t, err)

	pp := func(node *yaml.Node) error {
		// rename "bad-name" key to "fixed-name"
		if keyNode, _ := walky.GetKeyValue(node, walky.NewStringNode("bad-name")); keyNode != nil {
			keyNode.Value = "fixed-name"
		}

		// reset "bad-value" key to "fixed-value", under "good-name" key
		if valNode := walky.GetKey(node, "good-name"); valNode != nil {
			valNode.Value = "fixed-value"
		}

		// migrate "ok-name" value from string to list of strings
		if keyNode, valNode := walky.GetKeyValue(node, walky.NewStringNode("ok-name")); keyNode != nil {
			if valNode.Kind == yaml.ScalarNode {
				seqNode := walky.NewSequenceNode()
				require.NoError(t,
					walky.AppendNode(seqNode, walky.ShallowCopyNode(valNode)),
				)
				require.NoError(t,
					walky.AssignMapNode(node, keyNode, seqNode),
				)
			}
		}
		return nil
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

	err = fig.LoadConfigSource(&input, "test", &dest)
	assert.NoError(t, err)
	assert.Equal(t, want, dest)
}

func TestMergeMapWithCopy(t *testing.T) {
	type mss = map[string]string

	dest := struct {
		Map mss
	}{}

	src1 := struct {
		Map mss
	}{
		mss{
			"key": "value",
		},
	}

	src2 := struct {
		Map mss
	}{
		mss{
			"otherkey": "othervalue",
		},
	}

	err := Merge(&dest, &src1)
	require.NoError(t, err)
	assert.Equal(t, mss{"key": "value"}, dest.Map)

	err = Merge(&dest, &src2)
	require.NoError(t, err)
	assert.Equal(t, mss{"key": "value", "otherkey": "othervalue"}, dest.Map)

	// verify that src1 was unmodified
	assert.Equal(t, mss{"key": "value"}, src1.Map)
}

func TestMergeBoolString(t *testing.T) {
	src1 := struct {
		EnableThing BoolOption
	}{NewBoolOption(true)}

	src2 := map[string]interface{}{
		"enable-thing": "true",
	}

	dest := MakeMergeStruct(src1, src2)
	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

	expected := &struct {
		EnableThing BoolOption
	}{BoolOption{Source: NewSource("merge"), Defined: true, Value: true}}

	assert.Equal(t, expected, dest)
}

func TestMergeStringBool(t *testing.T) {
	src1 := struct {
		EnableThing StringOption
	}{NewStringOption("true")}

	src2 := map[string]interface{}{
		"enable-thing": true,
	}

	dest := MakeMergeStruct(src1, src2)
	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

	expected := &struct {
		EnableThing StringOption
	}{StringOption{Source: NewSource("merge"), Defined: true, Value: "true"}}

	assert.Equal(t, expected, dest)
}

func TestMergeStringFloat64(t *testing.T) {
	src1 := struct {
		SomeThing StringOption
	}{NewStringOption("true")}

	src2 := map[string]interface{}{
		"some-thing": 42.0,
	}

	dest := MakeMergeStruct(src1, src2)
	err := Merge(dest, src1)
	require.NoError(t, err)
	err = Merge(dest, src2)
	require.NoError(t, err)

	expected := &struct {
		SomeThing StringOption
	}{StringOption{Source: NewSource("merge"), Defined: true, Value: "42"}}

	assert.Equal(t, expected, dest)
}

func TestMergeDefaults(t *testing.T) {
	src1 := &struct {
		SomeThing StringOption
	}{NewStringOption("foo")}

	src2 := &struct {
		SomeThing StringOption
	}{NewStringOption("bar")}

	dest := MakeMergeStruct(src1, src2)
	err := Merge(dest, src1)
	require.NoError(t, err)

	expected := &struct {
		SomeThing StringOption
	}{StringOption{Source: NewSource("default"), Defined: true, Value: "foo"}}

	assert.Equal(t, expected, dest)

	err = Merge(dest, src2)
	require.NoError(t, err)

	assert.Equal(t, expected, dest)
}

// TestMergeCopySlices verifies when we merge a non-nil slice onto a nil-slice
// that the result is a copy of the original rather than direct reference
// assignment.  Otherwise we will get into conditions where we have multiple
// merged objects using the exact same reference to a slice where if we change
// one slice it modified all merged structs.
func TestMergeCopySlice(t *testing.T) {
	type stuffer = struct {
		Stuff []string
	}

	stuffers := []*stuffer{}
	common := &stuffer{Stuff: []string{"common"}}

	stuff1 := &stuffer{Stuff: nil}
	stuff2 := &stuffer{Stuff: nil}

	for _, stuff := range []*stuffer{stuff1, stuff2} {
		err := Merge(stuff, common)
		require.NoError(t, err)
		stuffers = append(stuffers, stuff)
	}

	assert.Equal(t, []string{"common"}, stuffers[0].Stuff)
	assert.Equal(t, []string{"common"}, stuffers[1].Stuff)

	stuffers[0].Stuff[0] = "updated"
	assert.Equal(t, []string{"updated"}, stuffers[0].Stuff)
	assert.Equal(t, []string{"common"}, stuffers[1].Stuff)
}

func TestMergeCopyArray(t *testing.T) {
	type stuffer = struct {
		Stuff [2]string
	}

	stuffers := []*stuffer{}
	common := &stuffer{Stuff: [2]string{"common"}}

	stuff1 := &stuffer{Stuff: [2]string{}}
	stuff2 := &stuffer{Stuff: [2]string{}}

	for _, stuff := range []*stuffer{stuff1, stuff2} {
		err := Merge(stuff, common)
		require.NoError(t, err)
		stuffers = append(stuffers, stuff)
	}

	assert.Equal(t, [2]string{"common", ""}, stuffers[0].Stuff)
	assert.Equal(t, [2]string{"common", ""}, stuffers[1].Stuff)

	stuffers[0].Stuff[0] = "updated"
	assert.Equal(t, [2]string{"updated", ""}, stuffers[0].Stuff)
	assert.Equal(t, [2]string{"common", ""}, stuffers[1].Stuff)
}

func TestListOfStructs(t *testing.T) {
	type myStruct struct {
		ID   string `yaml:"id"`
		Name string `yaml:"name"`
	}
	type myStructs []myStruct
	type data struct {
		Structs myStructs `yaml:"list"`
	}

	config := `
list:
  - id: abc
    name: def
  - id: foo
    name: bar
`
	expected := data{
		Structs: myStructs{
			{ID: "abc", Name: "def"},
			{ID: "foo", Name: "bar"},
		},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	dest := data{}
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = data{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestLoadConfigToNode(t *testing.T) {
	type SubData struct {
		Field yaml.Node `yaml:"field"`
	}
	type data struct {
		SubData `yaml:",inline"`
		List    []yaml.Node          `yaml:"list"`
		Map     map[string]yaml.Node `yaml:"map"`
		Stuff   yaml.Node            `yaml:"stuff"`
		Sub     SubData              `yaml:"sub"`
	}

	config := `
field: 123
list: [a, 99]
map:
  key1: abc
  key2: 123
stuff: {a: 1, b: 2}
sub:
  field: ghi
`
	expected := data{
		SubData: SubData{
			Field: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "123", Line: 2, Column: 8},
		},
		List: []yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a", Line: 3, Column: 8},
			{Kind: yaml.ScalarNode, Tag: "!!int", Value: "99", Line: 3, Column: 11},
		},
		Map: map[string]yaml.Node{
			"key1": {Kind: yaml.ScalarNode, Tag: "!!str", Value: "abc", Line: 5, Column: 9},
			"key2": {Kind: yaml.ScalarNode, Tag: "!!int", Value: "123", Line: 6, Column: 9},
		},
		Stuff: yaml.Node{
			Kind:  yaml.MappingNode,
			Tag:   "!!map",
			Style: yaml.FlowStyle,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a", Line: 7, Column: 9},
				{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1", Line: 7, Column: 12},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "b", Line: 7, Column: 15},
				{Kind: yaml.ScalarNode, Tag: "!!int", Value: "2", Line: 7, Column: 18},
			},
			Line:   7,
			Column: 8,
		},
		Sub: SubData{
			Field: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ghi", Line: 9, Column: 10},
		},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	dest := data{}
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

type UnmarshalInt int

func (t *UnmarshalInt) UnmarshalYAML(unmarshal func(any) error) error {
	var rawType string
	if err := unmarshal(&rawType); err != nil {
		return errors.WithStack(err)
	}
	switch strings.ToLower(rawType) {
	case "foo":
		*t = 1
	case "bar":
		*t = 2
	default:
		return errors.Errorf("Unknown unmarshal test value: %s", rawType)
	}
	return nil
}

func TestLoadConfigWithUnmarshalInt(t *testing.T) {
	type Property struct {
		Type UnmarshalInt  `yaml:"type"`
		Ptr  *UnmarshalInt `yaml:"ptr"`
	}
	type data struct {
		Properties []Property `yaml:"properties"`
	}

	config := `
properties:
  - type: foo
    ptr: foo
  - type: bar
    ptr: bar
`
	foo := UnmarshalInt(1)
	bar := UnmarshalInt(2)

	expected := data{
		Properties: []Property{
			{Type: foo, Ptr: &foo},
			{Type: bar, Ptr: &bar},
		},
	}

	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	dest := data{}
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = data{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

type UnmarshalString string

func (t *UnmarshalString) UnmarshalYAML(unmarshal func(any) error) error {
	var rawType any
	if err := unmarshal(&rawType); err != nil {
		return errors.WithStack(err)
	}
	switch c := rawType.(type) {
	case string:
		*t = UnmarshalString(strings.ToUpper(c))
	case int:
		*t = UnmarshalString(fmt.Sprint(c))
	default:
		panic(fmt.Sprintf("can't handle %T", c))
	}
	return nil
}

type UnmarshalStringList []UnmarshalString

// UnmarshalYAML will unmarshal a list of PortSpecs and return an error if any items
// could not be unmarshalled.
func (pl *UnmarshalStringList) UnmarshalYAML(unmarshal func(any) error) error {
	var ps []UnmarshalString
	if err := unmarshal(&ps); err != nil {
		return err
	}

	*pl = ps
	return nil
}

func TestLoadConfigWithUnmarshalString(t *testing.T) {
	type Property struct {
		Type UnmarshalString  `yaml:"type"`
		Ptr  *UnmarshalString `yaml:"ptr"`
	}
	type data struct {
		Properties []Property          `yaml:"properties"`
		Prop       Property            `yaml:"prop"`
		Str        UnmarshalString     `yaml:"str"`
		Ptr        *UnmarshalString    `yaml:"ptr"`
		Strs       []UnmarshalString   `yaml:"strs"`
		Ptrs       []*UnmarshalString  `yaml:"ptrs"`
		PtrStrs    *[]UnmarshalString  `yaml:"ptr-strs"`
		StrList    UnmarshalStringList `yaml:"str-list"`
	}

	config := `
properties:
  - type: foo
    ptr: foo
  - type: bar
    ptr: bar
prop:
  type: 123
  ptr: 123
ptr: a
str: a
strs: [a, b]
ptrs: [a, b]
str-list: [a, b]
ptr-strs: [a, b]
`
	foo := UnmarshalString("FOO")
	bar := UnmarshalString("BAR")
	baz := UnmarshalString("123")
	a := UnmarshalString("A")
	b := UnmarshalString("B")

	expected := data{
		Properties: []Property{
			{Type: foo, Ptr: &foo},
			{Type: bar, Ptr: &bar},
		},
		Prop:    Property{Type: baz, Ptr: &baz},
		Str:     a,
		Ptr:     &a,
		Strs:    []UnmarshalString{a, b},
		Ptrs:    []*UnmarshalString{&a, &b},
		StrList: UnmarshalStringList{a, b},
		PtrStrs: &[]UnmarshalString{a, b},
	}

	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	dest := data{}
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = data{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestLoadConfigWithSliceDups(t *testing.T) {
	type data struct {
		Strs    []UnmarshalString `yaml:"strs"`
		Simple  []string          `yaml:"simple"`
		Options ListStringOption  `yaml:"options"`
	}
	configs := []struct {
		Name string
		Body string
	}{{
		Name: "test",
		Body: `
strs: [a, b]
simple: [a, b]
options: [a, b]
`,
	}, {
		Name: "../test",
		Body: `
strs: [b ,c]
simple: [b, c]
options: [b, c]
`,
	}}
	expected := data{
		Strs:   []UnmarshalString{"A", "B", "C"},
		Simple: []string{"a", "b", "c"},
		Options: []StringOption{
			{tSrc("test", 4, 11), true, "a"},
			{tSrc("test", 4, 14), true, "b"},
			{tSrc("../test", 4, 14), true, "c"},
		},
	}
	sources := []ConfigSource{}
	for _, c := range configs {
		var node yaml.Node
		err := yaml.Unmarshal([]byte(c.Body), &node)
		require.NoError(t, err)
		sources = append(sources, ConfigSource{
			Config:   &node,
			Filename: c.Name,
		})
	}
	fig := newFigTreeFromEnv()
	got := data{}
	err := fig.LoadAllConfigSources(sources, &got)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestMapOfOptionLists(t *testing.T) {
	type data struct {
		Stuff map[string]ListStringOption `yaml:"stuff"`
	}

	config := `
stuff:
  foo:
    - abc
    - def
  bar:
    - ghi
    - jkl
`
	expected := data{
		Stuff: map[string]ListStringOption{
			"bar": {
				StringOption{tSrc("test", 7, 7), true, "ghi"},
				StringOption{tSrc("test", 8, 7), true, "jkl"},
			},
			"foo": {
				StringOption{tSrc("test", 4, 7), true, "abc"},
				StringOption{tSrc("test", 5, 7), true, "def"},
			},
		},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	dest := data{}
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestMapOfStructs(t *testing.T) {
	type myStruct struct {
		Name string
	}
	var dest map[string]myStruct

	config := `
foo:
  name: abc
bar:
  name: def
`
	expected := map[string]myStruct{
		"bar": {Name: "def"},
		"foo": {Name: "abc"},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = map[string]myStruct{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestZeroYAML(t *testing.T) {
	type myStruct struct {
		Name string
	}
	var dest map[string]myStruct

	config := `
foo:
bar:
`
	expected := map[string]myStruct{
		"bar": {},
		"foo": {},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = map[string]myStruct{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestBoolToStringOption(t *testing.T) {
	type myStruct struct {
		Name StringOption
	}
	var data map[string]myStruct

	// true/false will be parsed as `!!bool`
	// but we want to assign it to a StringOption
	config := `
foo:
  name: true
bar:
  name: false
`
	expected := map[string]myStruct{
		"bar": {Name: StringOption{tSrc("test", 5, 9), true, "false"}},
		"foo": {Name: StringOption{tSrc("test", 3, 9), true, "true"}},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &data)
	require.NoError(t, err)
	require.Equal(t, expected, data)
}

func TestBoolToString(t *testing.T) {
	type myStruct struct {
		Name string
	}
	var dest map[string]myStruct

	// true/false will be parsed as `!!bool`
	// but we want to assign it to a string
	config := `
foo:
  name: true
bar:
  name: false
`
	expected := map[string]myStruct{
		"bar": {Name: "false"},
		"foo": {Name: "true"},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = map[string]myStruct{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestIntToStringOption(t *testing.T) {
	var data map[string]StringOption

	config := `
a: 11
b: 11.1
c: 11.1.1
`
	expected := map[string]StringOption{
		"a": {tSrc("test", 2, 4), true, "11"},
		"b": {tSrc("test", 3, 4), true, "11.1"},
		"c": {tSrc("test", 4, 4), true, "11.1.1"},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &data)
	require.NoError(t, err)
	require.Equal(t, expected, data)
}

func TestIntToString(t *testing.T) {
	var dest map[string]string
	config := `
a: 11
b: 11.1
c: 11.1.1
`
	expected := map[string]string{
		"a": "11",
		"b": "11.1",
		"c": "11.1.1",
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = map[string]string{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestAssignToAnyOption(t *testing.T) {
	var data map[string]Option[any]
	config := `
a: foo
b: 12
c: 12.2
d: 12.2.2
`
	expected := map[string]Option[any]{
		"a": {tSrc("test", 2, 4), true, "foo"},
		"b": {tSrc("test", 3, 4), true, 12},
		"c": {tSrc("test", 4, 4), true, 12.2},
		"d": {tSrc("test", 5, 4), true, "12.2.2"},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &data)
	require.NoError(t, err)
	require.Equal(t, expected, data)
}

func TestAssignToAny(t *testing.T) {
	var dest map[string]any
	config := `
a: foo
b: 12
c: 12.2
d: 12.2.2
`
	expected := map[string]any{
		"a": "foo",
		"b": 12,
		"c": 12.2,
		"d": "12.2.2",
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = map[string]any{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestAssignInterfaceListToListStringOption(t *testing.T) {
	type data struct {
		MyList ListStringOption `yaml:"mylist"`
	}
	config := `
mylist: []
`
	expected := data{}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	dest := data{}
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.NoError(t, err)
	require.Equal(t, expected, dest)

	content, err := yaml.Marshal(dest)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	dest = data{}
	err = Merge(&dest, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, dest)
}

func TestAssignStringIntoList(t *testing.T) {
	type data struct {
		MyList ListStringOption `yaml:"mylist"`
	}
	config := `
mylist: foobar
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	dest := data{}
	err = fig.LoadConfigSource(&node, "test", &dest)
	require.Error(t, err)
}

func TestAssignStringToOptionPointer(t *testing.T) {
	type data struct {
		MyStr *StringOption `yaml:"my-str"`
	}
	config := `
my-str: abc
`
	expected := data{
		MyStr: &StringOption{tSrc("test", 2, 9), true, "abc"},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()

	got := data{MyStr: &StringOption{}}
	err = fig.LoadConfigSource(&node, "test", &got)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	got = data{}
	err = fig.LoadConfigSource(&node, "test", &got)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestAssignYAMLMergeMap(t *testing.T) {
	type data struct {
		MyMap    map[string]int `yaml:"my-map"`
		ExtraMap map[string]int `yaml:"extra-map"`
	}
	config := `
defs:
  - &common
    a: 1
    b: 2
  - &extra
    d: 4
    e: 5
my-map:
  <<: *common
  c: 3
extra-map:
  <<: [*common, *extra]
  c: 3
`
	expected := data{
		MyMap: map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
		},
		ExtraMap: map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
			"d": 4,
			"e": 5,
		},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()

	got := data{}
	err = fig.LoadConfigSource(&node, "test", &got)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestDecodeWithSource(t *testing.T) {
	StringifyValue = false
	defer func() {
		StringifyValue = true
	}()

	type data struct {
		MyMap    map[string]IntOption `yaml:"my-map"`
		ExtraMap map[string]IntOption `yaml:"extra-map"`
	}
	config := `
defs:
  - &common
    a: 1
    b: 2
    c:
  - &extra
    e: 4
    f: 5
    g:
my-map:
  <<: *common
  d: 3
extra-map:
  <<: [*common, *extra]
  d: 3
`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	if err != nil {
		panic(err)
	}
	fig := newFigTreeFromEnv()

	got := data{}
	err = fig.LoadConfigSource(&node, "test", &got)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = yaml.NewEncoder(&buf).Encode(&got)
	require.NoError(t, err)

	expected := `
	my-map:
	    a:
	        value: 1
	        source: test:4:8
	        defined: true
	    b:
	        value: 2
	        source: test:5:8
	        defined: true
	    c:
	        value: 0
	        source: test:6:7
	        defined: false
	    d:
	        value: 3
	        source: test:13:6
	        defined: true
	extra-map:
	    a:
	        value: 1
	        source: test:4:8
	        defined: true
	    b:
	        value: 2
	        source: test:5:8
	        defined: true
	    c:
	        value: 0
	        source: test:6:7
	        defined: false
	    d:
	        value: 3
	        source: test:16:6
	        defined: true
	    e:
	        value: 4
	        source: test:8:8
	        defined: true
	    f:
	        value: 5
	        source: test:9:8
	        defined: true
	    g:
	        value: 0
	        source: test:10:7
	        defined: false
`
	expected = strings.TrimLeftFunc(expected, unicode.IsSpace)
	expected = strings.ReplaceAll(expected, "\t", "")
	require.Equal(t, expected, buf.String())
}

func TestPreserveRawYAMLStrings(t *testing.T) {
	type data struct {
		MyMap map[string]string `yaml:"my-map"`
	}
	config := `
my-map:
  Prop1: "False"
  Prop2: False
  Prop3: 12
  Prop4: 12.3
`
	expected := data{
		MyMap: map[string]string{
			"Prop1": "False",
			"Prop2": "False",
			"Prop3": "12",
			"Prop4": "12.3",
		},
	}
	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()
	got := data{}
	err = fig.LoadConfigSource(&node, "test", &got)
	require.NoError(t, err)
	require.Equal(t, expected, got)

	content, err := yaml.Marshal(got)
	require.NoError(t, err)
	raw := map[string]any{}
	err = yaml.Unmarshal(content, &raw)
	require.NoError(t, err)

	got = data{}
	err = Merge(&got, &raw)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestMergeZeroOption(t *testing.T) {
	type testData struct {
		Data StringOption
	}
	dest := testData{
		Data: NewStringOption("default"),
	}
	err := Merge(&dest, &testData{})
	require.NoError(t, err)

	expected := testData{
		Data: NewStringOption("default"),
	}

	require.Equal(t, expected, dest)
}

func TestYAMLReferences(t *testing.T) {
	type stuffOptions struct {
		A StringOption
		B StringOption
		C StringOption
		D StringOption
	}
	type stuffStrings struct {
		A string
		B string
		C string
		D string
	}
	type data struct {
		Stuff1 stuffOptions      `yaml:"stuff1"`
		Stuff2 stuffStrings      `yaml:"stuff2"`
		Stuff3 map[string]string `yaml:"stuff3"`
	}

	config := `
defs:
 - &mystuff
   a: 1
   b: 2
 - &num 42
stuff1:
  <<: *mystuff
  c: 3
  d: *num
stuff2:
  <<: *mystuff
  c: 4
  d: *num
stuff3:
  <<: *mystuff
  c: 5
  d: *num
`
	expected := data{
		Stuff1: stuffOptions{
			A: StringOption{tSrc("test", 4, 7), true, "1"},
			B: StringOption{tSrc("test", 5, 7), true, "2"},
			C: StringOption{tSrc("test", 9, 6), true, "3"},
			D: StringOption{tSrc("test", 6, 4), true, "42"},
		},
		Stuff2: stuffStrings{
			A: "1",
			B: "2",
			C: "4",
			D: "42",
		},
		Stuff3: map[string]string{
			"a": "1",
			"b": "2",
			"c": "5",
			"d": "42",
		},
	}

	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()

	got := data{}
	err = fig.LoadConfigSource(&node, "test", &got)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestYAMLReferencesMaps(t *testing.T) {
	type stuffOptions struct {
		Map    MapStringOption
		Merged StringOption
		Extra  StringOption
	}
	type stuffMap struct {
		Map    map[string]string
		Merged string
		Extra  string
	}
	type data struct {
		Stuff1 stuffOptions `yaml:"stuff1"`
		Stuff2 stuffOptions `yaml:"stuff2"`
		Stuff3 stuffMap     `yaml:"stuff3"`
		Stuff4 stuffMap     `yaml:"stuff4"`
	}

	config := `
defs:
 - &map1
   map: {a: 1, b: 2} # this is ignored, dup key with merge site
   merged: "map1"
 - &map2
   map: {b: 3, c: 4, d: 5} # this is ignored, dup key with merge site
   merged: "map2"
stuff1:
  <<: *map1
  map: {b: 3, c: 4}
  extra: stuff1
stuff2:
  <<: [*map1, *map2]
  map: {b: 5, c: 6, e: 7}
  extra: stuff2
stuff3:
  <<: *map1
  map: {b: 3, c: 4}
  extra: stuff3
stuff4:
  <<: [*map1, *map2]
  map: {b: 5, c: 6, e: 7}
  extra: stuff4
`
	expected := data{
		Stuff1: stuffOptions{
			Map: MapStringOption{
				"b": {tSrc("test", 11, 12), true, "3"},
				"c": {tSrc("test", 11, 18), true, "4"},
			},
			Merged: StringOption{tSrc("test", 5, 12), true, "map1"},
			Extra:  StringOption{tSrc("test", 12, 10), true, "stuff1"},
		},
		Stuff2: stuffOptions{
			Map: MapStringOption{
				"b": {tSrc("test", 15, 12), true, "5"},
				"c": {tSrc("test", 15, 18), true, "6"},
				"e": {tSrc("test", 15, 24), true, "7"},
			},
			Merged: StringOption{tSrc("test", 5, 12), true, "map1"},
			Extra:  StringOption{tSrc("test", 16, 10), true, "stuff2"},
		},
		Stuff3: stuffMap{
			Map: map[string]string{
				"b": "3",
				"c": "4",
			},
			Merged: "map1",
			Extra:  "stuff3",
		},
		Stuff4: stuffMap{
			Map: map[string]string{
				"b": "5",
				"c": "6",
				"e": "7",
			},
			Merged: "map1",
			Extra:  "stuff4",
		},
	}

	var node yaml.Node
	err := yaml.Unmarshal([]byte(config), &node)
	require.NoError(t, err)
	fig := newFigTreeFromEnv()

	got := data{}
	err = fig.LoadConfigSource(&node, "test", &got)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestNullMerge(t *testing.T) {
	type data struct {
		Stuff MapStringOption
	}
	configs := []string{`
stuff:
  a:
`, `
stuff:
  a: 1
`}

	sources := []ConfigSource{}
	for i, config := range configs {
		var node yaml.Node
		err := yaml.Unmarshal([]byte(config), &node)
		require.NoError(t, err)
		sources = append(sources, ConfigSource{
			Config:   &node,
			Filename: "config" + strconv.Itoa(i),
		})
	}
	got := data{}
	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigSources(sources, &got)
	require.NoError(t, err)
	expected := data{
		Stuff: MapStringOption{
			"a": {tSrc("config1", 3, 6), true, "1"},
		},
	}
	require.Equal(t, expected, got)
}

func TestArrayAllowDupsOnPrimarySource(t *testing.T) {
	type data struct {
		Stuff1 []string         `yaml:"stuff1"`
		Stuff2 ListStringOption `yaml:"stuff2"`
	}

	configs := []string{`
stuff1: [a, b, a, a]
stuff2: [a, b, a, a]
`, `
stuff1: [a, b, c]
stuff2: [a, b, c]
`}

	sources := []ConfigSource{}
	for i, config := range configs {
		var node yaml.Node
		err := yaml.Unmarshal([]byte(config), &node)
		require.NoError(t, err)
		sources = append(sources, ConfigSource{
			Config:   &node,
			Filename: "config" + strconv.Itoa(i),
		})
	}
	got := data{}
	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigSources(sources, &got)
	require.NoError(t, err)
	expected := data{
		Stuff1: []string{"a", "b", "a", "a", "c"},
		Stuff2: ListStringOption{
			{tSrc("config0", 3, 10), true, "a"},
			{tSrc("config0", 3, 13), true, "b"},
			{tSrc("config0", 3, 16), true, "a"},
			{tSrc("config0", 3, 19), true, "a"},
			{tSrc("config1", 3, 16), true, "c"},
		},
	}
	require.Equal(t, expected, got)
}
