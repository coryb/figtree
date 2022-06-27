package figtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsExecConfigD3(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{tSrc("exec.yml[stdout]", 3, 5), true, "d3arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("exec.yml[stdout]", 4, 5), true, "d3arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("../exec.yml[stdout]", 3, 5), true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../exec.yml[stdout]", 4, 5), true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("../../exec.yml[stdout]", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../../exec.yml[stdout]", 4, 5), true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{tSrc("exec.yml[stdout]", 1, 7), true, "d3str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {tSrc("../../exec.yml[stdout]", 6, 9), true, "d1map1val0"},
			"key1": {tSrc("../exec.yml[stdout]", 6, 9), true, "d2map1val1"},
			"key2": {tSrc("exec.yml[stdout]", 6, 9), true, "d3map1val2"},
			"key3": {tSrc("exec.yml[stdout]", 7, 9), true, "d3map1val3"},
		},
		Int1:   IntOption{tSrc("exec.yml[stdout]", 8, 7), true, 333},
		Float1: Float32Option{tSrc("exec.yml[stdout]", 9, 9), true, 3.33},
		Bool1:  BoolOption{tSrc("exec.yml[stdout]", 10, 8), true, true},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("exec.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsExecConfigD2(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1/d2"))
	t.Cleanup(func() {
		_ = os.Chdir("../..")
	})

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{tSrc("exec.yml[stdout]", 3, 5), true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("exec.yml[stdout]", 4, 5), true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("../exec.yml[stdout]", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../exec.yml[stdout]", 4, 5), true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{tSrc("exec.yml[stdout]", 1, 7), true, "d2str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {tSrc("../exec.yml[stdout]", 6, 9), true, "d1map1val0"},
			"key1": {tSrc("exec.yml[stdout]", 6, 9), true, "d2map1val1"},
			"key2": {tSrc("exec.yml[stdout]", 7, 9), true, "d2map1val2"},
		},
		Int1:   IntOption{tSrc("exec.yml[stdout]", 8, 7), true, 222},
		Float1: Float32Option{tSrc("exec.yml[stdout]", 9, 9), true, 2.22},
		Bool1:  BoolOption{tSrc("exec.yml[stdout]", 10, 8), true, false},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("exec.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestOptionsExecConfigD1(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{tSrc("exec.yml[stdout]", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("exec.yml[stdout]", 4, 5), true, "d1arr1val2"})

	expected := TestOptions{
		String1:    StringOption{tSrc("exec.yml[stdout]", 1, 7), true, "d1str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {tSrc("exec.yml[stdout]", 6, 9), true, "d1map1val0"},
			"key1": {tSrc("exec.yml[stdout]", 7, 9), true, "d1map1val1"},
		},
		Int1:   IntOption{tSrc("exec.yml[stdout]", 8, 7), true, 111},
		Float1: Float32Option{tSrc("exec.yml[stdout]", 9, 9), true, 1.11},
		Bool1:  BoolOption{tSrc("exec.yml[stdout]", 10, 8), true, true},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("exec.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinExecConfigD3(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})

	arr1 := []string{}
	arr1 = append(arr1, "d3arr1val1")
	arr1 = append(arr1, "d3arr1val2")
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
		},
		Int1:   333,
		Float1: 3.33,
		Bool1:  true,
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("exec.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinExecConfigD2(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1/d2"))
	t.Cleanup(func() {
		_ = os.Chdir("../..")
	})

	arr1 := []string{}
	arr1 = append(arr1, "d2arr1val1")
	arr1 = append(arr1, "d2arr1val2")
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
		},
		Int1:   222,
		Float1: 2.22,
		// note this will be true from d1/exec.yml since the
		// d1/d2/exec.yml set it to false which is a zero value
		Bool1: true,
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("exec.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinExecConfigD1(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	arr1 := []string{}
	arr1 = append(arr1, "d1arr1val1")
	arr1 = append(arr1, "d1arr1val2")

	expected := TestBuiltin{
		String1:    "d1str1val1",
		LeaveEmpty: "",
		Array1:     arr1,
		Map1: map[string]string{
			"key0": "d1map1val0",
			"key1": "d1map1val1",
		},
		Int1:   111,
		Float1: 1.11,
		Bool1:  true,
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("exec.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}
