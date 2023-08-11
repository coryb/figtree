package figtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type Stringish struct {
	value string
}

// implement the Setter interface used to convert strings to option type
func (s *Stringish) Set(v string) error {
	s.value = v
	return nil
}

func TestCommandLine(t *testing.T) {
	type CommandLineOptions struct {
		Str1 StringOption          `yaml:"str1,omitempty"`
		Int1 IntOption             `yaml:"int1,omitempty"`
		Map1 MapStringOption       `yaml:"map1,omitempty"`
		Arr1 ListStringOption      `yaml:"arr1,omitempty"`
		Strs ListOption[Stringish] `yaml:"strs,omitempty"`
	}

	opts := CommandLineOptions{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	require.NoError(t, err)

	app := kingpin.New("test", "testing")
	app.Flag("str1", "Str1").SetValue(&opts.Str1)
	app.Flag("int1", "Int1").SetValue(&opts.Int1)
	app.Flag("map1", "Map1").SetValue(&opts.Map1)
	app.Flag("arr1", "Arr1").SetValue(&opts.Arr1)
	app.Flag("str", "Strs").SetValue(&opts.Strs)
	_, err = app.Parse([]string{"--int1", "999", "--map1", "k1=v1", "--map1", "k2=v2", "--arr1", "v1", "--arr1", "v2", "--str", "abc", "--str", "def"})
	require.NoError(t, err)

	arr1 := ListStringOption{}
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 3, 5), true, "d3arr1val1"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 4, 5), true, "d3arr1val2"})
	arr1 = append(arr1, StringOption{tSrc("figtree.yml", 5, 5), true, "dupval"})
	arr1 = append(arr1, StringOption{tSrc("../figtree.yml", 3, 5), true, "211"})
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
			"key1": {tSrc("../figtree.yml", 7, 9), true, "211"},
			"key2": {tSrc("figtree.yml", 7, 9), true, "d3map1val2"},
			"key3": {tSrc("figtree.yml", 8, 9), true, "d3map1val3"},
			"dup":  {tSrc("figtree.yml", 9, 9), true, "d3dupval"},
			"k1":   {NewSource("override"), true, "v1"},
			"k2":   {NewSource("override"), true, "v2"},
		},
		Arr1: arr1,
		Strs: ListOption[Stringish]{
			{NewSource("override"), true, Stringish{"abc"}},
			{NewSource("override"), true, Stringish{"def"}},
		},
	}

	require.Equal(t, expected, opts)
}

func TestCommandLineAny(t *testing.T) {
	type CommandLineOptions struct {
		Any1 Option[any]     `yaml:"any1,omitempty"`
		Map1 MapOption[any]  `yaml:"map1,omitempty"`
		Arr1 ListOption[any] `yaml:"arr1,omitempty"`
	}

	opts := CommandLineOptions{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	require.NoError(t, err)

	app := kingpin.New("test", "testing")
	app.Flag("any1", "Any1").SetValue(&opts.Any1)
	app.Flag("map1", "Map1").SetValue(&opts.Map1)
	app.Flag("arr1", "Arr1").SetValue(&opts.Arr1)
	_, err = app.Parse([]string{"--any1", "123", "--map1", "k1=v1", "--map1", "k2=v2", "--arr1", "v1", "--arr1", "v2"})
	require.NoError(t, err)

	arr1 := ListOption[any]{}
	arr1 = append(arr1, Option[any]{tSrc("figtree.yml", 3, 5), true, "d3arr1val1"})
	arr1 = append(arr1, Option[any]{tSrc("figtree.yml", 4, 5), true, "d3arr1val2"})
	arr1 = append(arr1, Option[any]{tSrc("figtree.yml", 5, 5), true, "dupval"})
	arr1 = append(arr1, Option[any]{tSrc("../figtree.yml", 3, 5), true, 211})
	arr1 = append(arr1, Option[any]{tSrc("../figtree.yml", 4, 5), true, "d2arr1val2"})
	arr1 = append(arr1, Option[any]{tSrc("../../figtree.yml", 3, 5), true, "d1arr1val1"})
	arr1 = append(arr1, Option[any]{tSrc("../../figtree.yml", 4, 5), true, "d1arr1val2"})
	arr1 = append(arr1, Option[any]{NewSource("override"), true, "v1"})
	arr1 = append(arr1, Option[any]{NewSource("override"), true, "v2"})

	expected := CommandLineOptions{
		Any1: Option[any]{NewSource("override"), true, "123"},
		Map1: map[string]Option[any]{
			"key0": {tSrc("../../figtree.yml", 7, 9), true, "d1map1val0"},
			"key1": {tSrc("../figtree.yml", 7, 9), true, 211},
			"key2": {tSrc("figtree.yml", 7, 9), true, "d3map1val2"},
			"key3": {tSrc("figtree.yml", 8, 9), true, "d3map1val3"},
			"dup":  {tSrc("figtree.yml", 9, 9), true, "d3dupval"},
			"k1":   {NewSource("override"), true, "v1"},
			"k2":   {NewSource("override"), true, "v2"},
		},
		Arr1: arr1,
	}

	require.Equal(t, expected, opts)
}
