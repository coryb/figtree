package figtree

import (
	"os"
	"testing"

	yaml "gopkg.in/coryb/yaml.v2"

	"github.com/stretchr/testify/assert"
)

func TestOptionsMarshall(t *testing.T) {
	opts := TestOptions{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")
	err := LoadConfig("figtree.yml", &opts)
	assert.Nil(t, err)

	got, err := yaml.Marshal(&opts)
	assert.Nil(t, err)

	expected := `str1: d3str1val1
arr1:
- d3arr1val1
- d3arr1val2
- d2arr1val1
- d2arr1val2
- d1arr1val1
- d1arr1val2
map1:
  key0: d1map1val0
  key1: d2map1val1
  key2: d3map1val2
  key3: d3map1val3
int1: 333
float1: 3.33
bool1: true
`
	assert.Equal(t, expected, string(got))
}
