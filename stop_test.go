package figtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionsStopConfigD3(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"stop.yml:3:5", true, "d3arr1val1"})
	arr1 = append(arr1, StringOption{"stop.yml:4:5", true, "d3arr1val2"})
	arr1 = append(arr1, StringOption{"../stop.yml:5:5", true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{"../stop.yml:6:5", true, "d2arr1val2"})

	expected := TestOptions{
		String1:    StringOption{"stop.yml:1:7", true, "d3str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key1": {"../stop.yml:8:9", true, "d2map1val1"},
			"key2": {"stop.yml:6:9", true, "d3map1val2"},
			"key3": {"stop.yml:7:9", true, "d3map1val3"},
		},
		Int1:   IntOption{"stop.yml:8:7", true, 333},
		Float1: Float32Option{"stop.yml:9:9", true, 3.33},
		Bool1:  BoolOption{"stop.yml:10:8", true, true},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("stop.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsStopConfigD2(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"stop.yml:5:5", true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{"stop.yml:6:5", true, "d2arr1val2"})

	expected := TestOptions{
		String1:    StringOption{"stop.yml:3:7", true, "d2str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key1": {"stop.yml:8:9", true, "d2map1val1"},
			"key2": {"stop.yml:9:9", true, "d2map1val2"},
		},
		Int1:   IntOption{"stop.yml:10:7", true, 222},
		Float1: Float32Option{"stop.yml:11:9", true, 2.22},
		Bool1:  BoolOption{"stop.yml:12:8", true, false},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("stop.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinStopConfigD3(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")

	arr1 := []string{}
	arr1 = append(arr1, "d3arr1val1")
	arr1 = append(arr1, "d3arr1val2")
	arr1 = append(arr1, "d2arr1val1")
	arr1 = append(arr1, "d2arr1val2")

	expected := TestBuiltin{
		String1:    "d3str1val1",
		LeaveEmpty: "",
		Array1:     arr1,
		Map1: map[string]string{
			"key1": "d2map1val1",
			"key2": "d3map1val2",
			"key3": "d3map1val3",
		},
		Int1:   333,
		Float1: 3.33,
		Bool1:  true,
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("stop.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinStopConfigD2(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	arr1 := []string{}
	arr1 = append(arr1, "d2arr1val1")
	arr1 = append(arr1, "d2arr1val2")

	expected := TestBuiltin{
		String1:    "d2str1val1",
		LeaveEmpty: "",
		Array1:     arr1,
		Map1: map[string]string{
			"key1": "d2map1val1",
			"key2": "d2map1val2",
		},
		Int1:   222,
		Float1: 2.22,
		Bool1:  false,
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("stop.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}
