package figtree

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func TestCommandLine(t *testing.T) {
	type CommandLineOptions struct {
		Str1 StringOption `yaml:"str1,omitempty"`
		Int1 IntOption    `yaml:"int1,omitempty"`
	}

	opts := CommandLineOptions{}
	os.Chdir("d1/d2/d3")
	defer os.Chdir("../../..")

	err := LoadAllConfigs("figtree.yml", &opts)
	assert.Nil(t, err)

	app := kingpin.New("test", "testing")
	app.Flag("str1", "Str1").SetValue(&opts.Str1)
	app.Flag("int1", "Int1").SetValue(&opts.Int1)
	_, err = app.Parse([]string{"--int1", "999"})
	assert.Nil(t, err)

	expected := CommandLineOptions{
		Str1: StringOption{"figtree.yml", true, "d3str1val1"},
		Int1: IntOption{"override", true, 999},
	}

	assert.Equal(t, expected, opts)
}
