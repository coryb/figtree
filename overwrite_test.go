package figtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func init() {
	StringifyValue = false
}

func TestOptionsOverwriteConfigD3(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"../overwrite.yml:8:5", true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{"../overwrite.yml:9:5", true, "d2arr1val2"})

	expected := TestOptions{
		String1:    StringOption{"../overwrite.yml:6:7", true, "d2str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {"../../overwrite.yml:11:9", true, "d1map1val0"},
			"key1": {"../../overwrite.yml:12:9", true, "d1map1val1"},
		},
		Int1:   IntOption{"../../overwrite.yml:13:7", true, 111},
		Float1: Float32Option{"../../overwrite.yml:14:9", true, 1.11},
		Bool1:  BoolOption{"../overwrite.yml:15:8", true, false},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("overwrite.yml", &opts)
	require.Nil(t, err)
	require.Exactly(t, expected, opts)
}

func TestOptionsOverwriteConfigD2(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	arr1 := []StringOption{}
	arr1 = append(arr1, StringOption{"overwrite.yml:8:5", true, "d2arr1val1"})
	arr1 = append(arr1, StringOption{"overwrite.yml:9:5", true, "d2arr1val2"})

	expected := TestOptions{
		String1:    StringOption{"overwrite.yml:6:7", true, "d2str1val1"},
		LeaveEmpty: StringOption{},
		Array1:     arr1,
		Map1: map[string]StringOption{
			"key0": {"../overwrite.yml:11:9", true, "d1map1val0"},
			"key1": {"../overwrite.yml:12:9", true, "d1map1val1"},
		},
		Int1:   IntOption{"../overwrite.yml:13:7", true, 111},
		Float1: Float32Option{"../overwrite.yml:14:9", true, 1.11},
		Bool1:  BoolOption{"overwrite.yml:15:8", true, false},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("overwrite.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinOverwriteConfigD3(t *testing.T) {
	opts := TestBuiltin{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")

	arr1 := []string{}
	arr1 = append(arr1, "d2arr1val1")
	arr1 = append(arr1, "d2arr1val2")

	expected := TestBuiltin{
		String1:    "d2str1val1",
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
	err := fig.LoadAllConfigs("overwrite.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

func TestBuiltinOverwriteConfigD2(t *testing.T) {
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
			"key0": "d1map1val0",
			"key1": "d1map1val1",
		},
		Int1:   111,
		Float1: 1.11,
		Bool1:  false,
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("overwrite.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

type TestArray struct {
	IntArr      [2]int    `yaml:"intArr"`
	PartialInt  [2]int    `yaml:"partialInt"`
	TooManyInt  [2]int    `yaml:"tooManyInt"`
	StrArr      [2]string `yaml:"strArr"`
	ToOverwrite [2]string `yaml:"toOverwrite"`
}

func TestBuiltinOverwriteArrayD2(t *testing.T) {
	opts := TestArray{}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	expected := TestArray{
		IntArr:      [2]int{1, 2},
		PartialInt:  [2]int{1, 0},
		TooManyInt:  [2]int{1, 2},
		StrArr:      [2]string{"abc", "def"},
		ToOverwrite: [2]string{"c", "d"},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("array.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

type TestOptionsArray struct {
	IntArr      [2]IntOption    `yaml:"intArr"`
	PartialInt  [2]IntOption    `yaml:"partialInt"`
	TooManyInt  [2]IntOption    `yaml:"tooManyInt"`
	StrArr      [2]StringOption `yaml:"strArr"`
	ToOverwrite [2]StringOption `yaml:"toOverwrite"`
}

func TestOptionsOverwriteArrayD2(t *testing.T) {
	opts := TestOptionsArray{}
	os.Chdir("d1/d2")
	defer os.Chdir("../..")

	expected := TestOptionsArray{
		IntArr: [2]IntOption{
			{"array.yml:1:10", true, 1},
			{"array.yml:1:12", true, 2},
		},
		PartialInt: [2]IntOption{
			{"array.yml:2:14", true, 1},
			{"", false, 0},
		},
		TooManyInt: [2]IntOption{
			{"array.yml:3:14", true, 1},
			{"array.yml:3:16", true, 2},
		},
		StrArr: [2]StringOption{
			{"array.yml:4:10", true, "abc"},
			{"array.yml:4:15", true, "def"},
		},
		ToOverwrite: [2]StringOption{
			{"../array.yml:5:15", true, "c"},
			{"../array.yml:5:17", true, "d"},
		},
	}

	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("array.yml", &opts)
	assert.Nil(t, err)
	assert.Exactly(t, expected, opts)
}

// auto-upgrade:
//   enabled: true

func TestOverwritePartialStruct(t *testing.T) {
	type AutoUpgrade struct {
		Channel StringOption `yaml:"channel"`
		Enabled BoolOption   `yaml:"enabled"`
	}
	type data struct {
		AutoUpgrade AutoUpgrade `yaml:"auto-upgrade"`
	}
	configs := []struct {
		Name string
		Body string
	}{{
		Name: "test",
		Body: `
auto-upgrade:
  enabled: true
`,
	}, {
		Name: "../test",
		Body: `
config: {overwrite: [auto-upgrade]}
auto-upgrade:
  enabled: false
  channel: stable
`,
	}}
	expected := data{
		AutoUpgrade: AutoUpgrade{
			Channel: StringOption{"../test:5:12", true, "stable"},
			Enabled: BoolOption{"../test:4:12", true, false},
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
