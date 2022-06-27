package figtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func TestCommandLine(t *testing.T) {
	type CommandLineOptions struct {
		Str1 StringOption     `yaml:"str1,omitempty"`
		Int1 IntOption        `yaml:"int1,omitempty"`
		Map1 MapStringOption  `yaml:"map1,omitempty"`
		Arr1 ListStringOption `yaml:"arr1,omitempty"`
	}

	opts := CommandLineOptions{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)

	app := kingpin.New("test", "testing")
	app.Flag("str1", "Str1").SetValue(&opts.Str1)
	app.Flag("int1", "Int1").SetValue(&opts.Int1)
	app.Flag("map1", "Map1").SetValue(&opts.Map1)
	app.Flag("arr1", "Arr1").SetValue(&opts.Arr1)
	_, err = app.Parse([]string{"--int1", "999", "--map1", "k1=v1", "--map1", "k2=v2", "--arr1", "v1", "--arr1", "v2"})
	assert.Nil(t, err)

	arr1 := ListStringOption{}
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 3, 5), true, "d3arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 4, 5), true, "d3arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 5, 5), true, "dupval"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 3, 5), true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 4, 5), true, "d2arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("../../figtree.yml", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("../../figtree.yml", 4, 5), true, "d1arr1val2"})
	arr1 = append(arr1, StringOption{NewSource("override"), true, "v1"})
	arr1 = append(arr1, StringOption{NewSource("override"), true, "v2"})

	expected := CommandLineOptions{
		Str1: StringOption{tSrc("figtree.yml", 1, 7), true, "d3str1val1"},
		Int1: IntOption{NewSource("override"), true, 999},
		Map1: map[string]StringOption{
			"key0": {tSrc("../../figtree.yml", 7, 9), true, "d1map1val0"},
			"key1": {tSrc("../figtree.yml", 7, 9), true, "d2map1val1"},
			"key2": {tSrc("figtree.yml", 7, 9), true, "d3map1val2"},
			"key3": {tSrc("figtree.yml", 8, 9), true, "d3map1val3"},
			"dup":  {tSrc("figtree.yml", 9, 9), true, "d3dupval"},
			"k1":   {NewSource("override"), true, "v1"},
			"k2":   {NewSource("override"), true, "v2"},
		},
		Arr1: arr1,
	}

	assert.Equal(t, expected, opts)
}
