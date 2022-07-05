package figtree

import (
	"encoding/json"
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"
)

func TestOptionInterface(t *testing.T) {
	f := func(_ option) bool {
		return true
	}

	assert.True(t, f(&BoolOption{}))
	assert.True(t, f(&ByteOption{}))
	assert.True(t, f(&Complex128Option{}))
	assert.True(t, f(&Complex64Option{}))
	assert.True(t, f(&ErrorOption{}))
	assert.True(t, f(&Float32Option{}))
	assert.True(t, f(&Float64Option{}))
	assert.True(t, f(&IntOption{}))
	assert.True(t, f(&Int16Option{}))
	assert.True(t, f(&Int32Option{}))
	assert.True(t, f(&Int64Option{}))
	assert.True(t, f(&Int8Option{}))
	assert.True(t, f(&RuneOption{}))
	assert.True(t, f(&StringOption{}))
	assert.True(t, f(&UintOption{}))
	assert.True(t, f(&Uint16Option{}))
	assert.True(t, f(&Uint32Option{}))
	assert.True(t, f(&Uint64Option{}))
	assert.True(t, f(&Uint8Option{}))
	assert.True(t, f(&UintptrOption{}))
}

func TestStringOptionYAML(t *testing.T) {
	s := ""
	err := yaml.Unmarshal([]byte(`""`), &s)
	assert.NoError(t, err)
	assert.Equal(t, s, "")

	type testType struct {
		String StringOption `yaml:"string,omitempty"`
	}
	tt := testType{}

	err = yaml.Unmarshal([]byte(`string: ""`), &tt)
	assert.NoError(t, err)
	assert.Equal(t, StringOption{Source: tSrc("yaml", 1, 9), Value: "", Defined: true}, tt.String)

	tt = testType{}
	err = yaml.Unmarshal([]byte(`string: "value"`), &tt)
	assert.NoError(t, err)
	assert.Equal(t, StringOption{Source: tSrc("yaml", 1, 9), Value: "value", Defined: true}, tt.String)
}

func TestStringOptionJSON(t *testing.T) {
	type testType struct {
		String StringOption `json:"string,omitempty"`
	}
	tt := testType{}

	err := json.Unmarshal([]byte(`{"string": ""}`), &tt)
	assert.NoError(t, err)
	assert.Equal(t, StringOption{Source: NewSource("json"), Value: "", Defined: true}, tt.String)

	tt = testType{}
	err = json.Unmarshal([]byte(`{"string": "value"}`), &tt)
	assert.NoError(t, err)
	assert.Equal(t, StringOption{Source: NewSource("json"), Value: "value", Defined: true}, tt.String)
}

func TestBoolOptionYAML(t *testing.T) {
	type testType struct {
		Bool BoolOption `yaml:"bool,omitempty"`
	}
	tt := testType{}

	err := yaml.Unmarshal([]byte(`bool: true`), &tt)
	assert.NoError(t, err)
	assert.Equal(t, BoolOption{Source: tSrc("yaml", 1, 7), Value: true, Defined: true}, tt.Bool)

	tt = testType{}
	err = yaml.Unmarshal([]byte(`bool: false`), &tt)
	assert.NoError(t, err)
	assert.Equal(t, BoolOption{Source: tSrc("yaml", 1, 7), Value: false, Defined: true}, tt.Bool)

	tt = testType{
		Bool: NewBoolOption(true),
	}
	err = yaml.Unmarshal([]byte(`bool: false`), &tt)
	assert.NoError(t, err)
	assert.Equal(t, BoolOption{Source: tSrc("yaml", 1, 7), Value: false, Defined: true}, tt.Bool)
}
