package figtree

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	yaml "gopkg.in/coryb/yaml.v2"
	logging "gopkg.in/op/go-logging.v1"

	"github.com/stretchr/testify/assert"
)

func init() {
	StringifyValue = false
	logging.SetLevel(logging.NOTICE, "")
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

	err := LoadAllConfigs("figtree.yml", &opts)
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

	err := LoadAllConfigs("figtree.yml", &opts)
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

	err := LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsCorrupt(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1")
	defer os.Chdir("..")

	err := LoadAllConfigs("corrupt.yml", &opts)
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

	err := LoadAllConfigs("figtree.yml", &opts)
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

	err := LoadAllConfigs("figtree.yml", &opts)
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

	err := LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinCorrupt(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1")
	defer os.Chdir("..")

	err := LoadAllConfigs("corrupt.yml", &opts)
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

	err := LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestMergeMapWithStruct(t *testing.T) {
	dest := map[string]interface{}{
		"mapkey": "mapval1",
		"map": map[string]interface{}{
			"mapkey": "mapval2",
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

	m := &merger{}
	m.mergeStructs(reflect.ValueOf(&dest), reflect.ValueOf(&src))

	expected := map[string]interface{}{
		"mapkey":       "mapval1",
		"struct-field": "field1",
		"map": map[string]interface{}{
			"mapkey":       "mapval2",
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
			"mapkey": "mapval2",
		},
	}

	m := &merger{}
	m.mergeStructs(reflect.ValueOf(&dest), reflect.ValueOf(&src))

	expected := struct {
		StructField string
		Mapkey      string
		Map         struct {
			StructField string
			Mapkey      string
		}
	}{
		StructField: "field1",
		Mapkey:      "mapval1",
		Map: struct {
			StructField string
			Mapkey      string
		}{
			StructField: "field2",
			Mapkey:      "mapval2",
		},
	}
	assert.Equal(t, expected, dest)
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
	}{}

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
			StringOption{"merge", true, "abc"},
			StringOption{"merge", true, "def"},
		},
	}
	assert.Equal(t, expected, dest)
}

func TestMergeMapWithStructUsingListOptions(t *testing.T) {
	dest := map[string]interface{}{
		"strings": []string{},
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

	m := &merger{}
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

	m := &merger{}
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
