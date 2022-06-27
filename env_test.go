package figtree

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOptionsEnv(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	StringifyValue = true
	defer func() {
		StringifyValue = false
	}()

	os.Clearenv()

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)

	got := []string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "FIGTREE_") {
			got = append(got, env)
		}
	}

	sort.StringSlice(got).Sort()

	expected := []string{
		"FIGTREE_ARRAY_1=[\"d1arr1val1\",\"d1arr1val2\",\"dupval\"]",
		"FIGTREE_BOOL_1=true",
		"FIGTREE_FLOAT_1=1.11",
		"FIGTREE_INT_1=111",
		"FIGTREE_MAP_1={\"dup\":\"d1dupval\",\"key0\":\"d1map1val0\",\"key1\":\"d1map1val1\"}",
		"FIGTREE_STRING_1=d1str1val1",
	}

	assert.Equal(t, expected, got)
}

func TestOptionsNamedEnv(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	StringifyValue = true
	defer func() {
		StringifyValue = false
	}()

	os.Clearenv()

	fig := newFigTreeFromEnv(WithEnvPrefix("TEST"))

	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)

	got := []string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "FIGTREE_") || strings.HasPrefix(env, "TEST_") {
			got = append(got, env)
		}
	}

	sort.StringSlice(got).Sort()

	expected := []string{
		"TEST_ARRAY_1=[\"d1arr1val1\",\"d1arr1val2\",\"dupval\"]",
		"TEST_BOOL_1=true",
		"TEST_FLOAT_1=1.11",
		"TEST_INT_1=111",
		"TEST_MAP_1={\"dup\":\"d1dupval\",\"key0\":\"d1map1val0\",\"key1\":\"d1map1val1\"}",
		"TEST_STRING_1=d1str1val1",
	}

	assert.Equal(t, expected, got)
}

func TestBuiltinEnv(t *testing.T) {
	opts := TestBuiltin{}
	require.NoError(t, os.Chdir("d1"))
	t.Cleanup(func() {
		_ = os.Chdir("..")
	})

	os.Clearenv()

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)

	got := []string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "FIGTREE_") {
			got = append(got, env)
		}
	}

	sort.StringSlice(got).Sort()

	expected := []string{
		"FIGTREE_ARRAY_1=[\"d1arr1val1\",\"d1arr1val2\",\"dupval\"]",
		"FIGTREE_BOOL_1=true",
		"FIGTREE_FLOAT_1=1.11",
		"FIGTREE_INT_1=111",
		"FIGTREE_LEAVE_EMPTY=",
		"FIGTREE_MAP_1={\"dup\":\"d1dupval\",\"key0\":\"d1map1val0\",\"key1\":\"d1map1val1\"}",
		"FIGTREE_STRING_1=d1str1val1",
	}

	assert.Equal(t, expected, got)
}

func TestTag(t *testing.T) {
	os.Clearenv()

	dest := struct {
		DefaultEnv  string `yaml:"default-env"`
		OverrideEnv string `yaml:"override-env" figtree:"OVERRIDE_ENV"`
		NoEnv       string `yaml:"no-env" figtree:"-"`
		MultiEnv    string `yaml:"multi-env" figtree:"MULTIA;MULTIB"`
	}{}

	var node yaml.Node
	err := yaml.Unmarshal([]byte(`
default-env: abc
override-env: def
no-env: ghi
multi-env: jkl
`), &node)
	assert.NoError(t, err)

	fig := newFigTreeFromEnv()
	err = fig.LoadConfigSource(&node, "test", &dest)
	assert.NoError(t, err)

	got := os.Environ()
	sort.StringSlice(got).Sort()

	expected := []string{
		"FIGTREE_DEFAULT_ENV=abc",
		"FIGTREE_MULTIA=jkl",
		"FIGTREE_MULTIB=jkl",
		"FIGTREE_OVERRIDE_ENV=def",
	}

	assert.Equal(t, expected, got)
}
