//go:generate genny -in=$GOFILE -out=gen-$GOFILE gen "RawType=BUILTINS"

package figtree

import (
	"encoding/json"
	"fmt"

	"github.com/cheekybits/genny/generic"
)

type RawType generic.Type

type RawTypeOption struct {
	Source  string
	Defined bool
	Value   RawType
}

func NewRawTypeOption(dflt RawType) RawTypeOption {
	return RawTypeOption{
		Source:  "default",
		Defined: false,
		Value:   dflt,
	}
}

func (o *RawTypeOption) IsDefined() bool {
	return o.Defined
}

func (o *RawTypeOption) SetSource(source string) {
	o.Source = source
}

func (o *RawTypeOption) GetValue() interface{} {
	return o.Value
}

// This is useful with kingpin option parser
func (o *RawTypeOption) Set(s string) error {
	err := convertString(s, &o.Value)
	if err != nil {
		return err
	}
	o.Source = "override"
	o.Defined = true
	return nil
}

func (o *RawTypeOption) SetValue(v interface{}) error {
	if val, ok := v.(RawType); ok {
		o.Value = val
		o.Defined = true
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", v, o.Value, v)
}

func (o *RawTypeOption) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&o.Value); err == nil {
		o.Defined = true
	} else {
		return err
	}

	return nil
}

func (o RawTypeOption) MarshalYAML() (interface{}, error) {
	return o.Value, nil
}

func (o RawTypeOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// String is required for kingpin to generate usage with this datatype
func (o RawTypeOption) String() string {
	if StringifyValue {
		return fmt.Sprintf("%v", o.Value)
	}
	return fmt.Sprintf("{Source:%s Defined:%t Value:%v}", o.Source, o.Defined, o.Value)
}

type MapRawTypeOption map[string]RawTypeOption

// Set is required for kingpin interfaces to allow command line params
// to be set to our map datatype
func (o *MapRawTypeOption) Set(value string) error {
	parts := stringMapRegex.Split(value, 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected KEY=VALUE got '%s'", value)
	}
	val := RawTypeOption{}
	val.Set(parts[1])
	(*o)[parts[0]] = val
	return nil
}

// IsCumulative is required for kingpin interfaces to allow multiple values
// to be set on the data structure.
func (o *MapRawTypeOption) IsCumulative() bool {
	return true
}

// String is required for kingpin to generate usage with this datatype
func (o MapRawTypeOption) String() string {
	return fmt.Sprintf("%v", map[string]RawTypeOption(o))
}
