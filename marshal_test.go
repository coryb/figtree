package figtree

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func yamlMarshal(opts interface{}) (string, error) {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	err := enc.Encode(opts)
	return buf.String(), err
}

func TestOptionsMarshalYAML(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})
	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)

	StringifyValue = true
	defer func() {
		StringifyValue = false
	}()
	got, err := yamlMarshal(&opts)
	assert.NoError(t, err)

	expected := `str1: d3str1val1
arr1:
  - d3arr1val1
  - d3arr1val2
  - dupval
  - "211"
  - d2arr1val2
  - d1arr1val1
  - d1arr1val2
map1:
  dup: d3dupval
  key0: d1map1val0
  key1: "211"
  key2: d3map1val2
  key3: d3map1val3
int1: 333
float1: 3.33
bool1: true
`
	assert.Equal(t, expected, got)
}

func TestOptionsMarshalJSON(t *testing.T) {
	opts := TestOptions{}
	require.NoError(t, os.Chdir("d1/d2/d3"))
	t.Cleanup(func() {
		_ = os.Chdir("../../..")
	})
	fig := newFigTreeFromEnv()
	err := fig.LoadAllConfigs("figtree.yml", &opts)
	assert.NoError(t, err)

	StringifyValue = true
	defer func() {
		StringifyValue = false
	}()
	got, err := json.Marshal(&opts)
	assert.NoError(t, err)
	// note that "leave-empty" is serialized even though "omitempty" tag is set
	// this is because json always assumes structs are not empty and there
	// is no interface to override this behavior
	expected := `{"str1":"d3str1val1","leave-empty":"","arr1":["d3arr1val1","d3arr1val2","dupval","211","d2arr1val2","d1arr1val1","d1arr1val2"],"map1":{"dup":"d3dupval","key0":"d1map1val0","key1":"211","key2":"d3map1val2","key3":"d3map1val3"},"int1":333,"float1":3.33,"bool1":true}`
	assert.Equal(t, expected, string(got))
}
