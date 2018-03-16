package figtree

import (
	"os"
	"reflect"
	"testing"

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
	assert.Equal(t, struct{ Mapkey string }{"mapval2"}, reflect.ValueOf(got).Elem().FieldByName("Map").Interface())
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
	assert.Equal(t, &struct{ Mapkey string }{"mapval1"}, got)

	got = MakeMergeStruct(s, input)
	Merge(got, &s)
	assert.Equal(t, &struct{ Mapkey string }{"mapval2"}, got)
}
