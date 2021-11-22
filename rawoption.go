//go:generate genny -in=$GOFILE -out=gen-$GOFILE gen "RawType=BUILTINS"

package figtree

import (
	"encoding/json"
	"fmt"

	"github.com/cheekybits/genny/generic"
)

// RawType is the generic type for go generate to use
type RawType generic.Type

// RawTypeOption hold data for configuration fields of type RawType
type RawTypeOption struct {
	Source  string
	Defined bool
	Value   RawType
}

// NewRawTypeOption returns a default configuration object of type RawType
func NewRawTypeOption(dflt RawType) RawTypeOption {
	return RawTypeOption{
		Source:  "default",
		Defined: true,
		Value:   dflt,
	}
}

// IsDefined returns if the option has been defined (things returned from
// NewRawTypeOption are defined by default)
func (o RawTypeOption) IsDefined() bool {
	return o.Defined
}

// SetSource allows setting the config file source path for the option
func (o *RawTypeOption) SetSource(source string) {
	o.Source = source
}

// GetSource returns the config file source path for the option
func (o *RawTypeOption) GetSource() string {
	return o.Source
}

// GetValue returns the raw value (type RawType) of the option
func (o RawTypeOption) GetValue() interface{} {
	return o.Value
}

// Set allows setting the value from a string.  It will be parsed from a string
// into the RawType.  This is useful with kingpin option parser
func (o *RawTypeOption) Set(s string) error {
	err := convertString(s, &o.Value)
	if err != nil {
		return err
	}
	o.Source = "override"
	o.Defined = true
	return nil
}

// WriteAnswer will assign the option value from the value provided during
// a prompt.  This is useful with survey prompting library
func (o *RawTypeOption) WriteAnswer(name string, value interface{}) error {
	if v, ok := value.(RawType); ok {
		o.Value = v
		o.Defined = true
		o.Source = "prompt"
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", value, o.Value, value)
}

// SetValue will set the option value from the provided interface. The interface
// value must be of type RawType.
func (o *RawTypeOption) SetValue(v interface{}) error {
	if val, ok := v.(RawType); ok {
		o.Value = val
		o.Defined = true
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", v, o.Value, v)
}

// UnmarshalYAML will populate the option from the parsed results of the
// yaml unmarshaller.
func (o *RawTypeOption) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&o.Value); err != nil {
		return err
	}
	o.Source = "yaml"
	o.Defined = true
	return nil
}

// UnmarshalJSON will populate the option from the parsed results for the
// json unmarshaller.
func (o *RawTypeOption) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &o.Value); err != nil {
		return err
	}
	o.Source = "json"
	o.Defined = true
	return nil
}

// MarshalYAML will convert the option to the RawType value when marshalling
// the data structure.
func (o RawTypeOption) MarshalYAML() (interface{}, error) {
	if StringifyValue {
		return o.Value, nil
	}
	// need a copy of this struct without the MarshalYAML interface attached
	return struct {
		Value   RawType
		Source  string
		Defined bool
	}{
		Value:   o.Value,
		Source:  o.Source,
		Defined: o.Defined,
	}, nil
}

// MarshalJSON will convert the option to the RawType value when marshalling
// the data structure.
func (o RawTypeOption) MarshalJSON() ([]byte, error) {
	if StringifyValue {
		return json.Marshal(o.Value)
	}
	// need a copy of this struct without the MarshalJSON interface attached
	return json.Marshal(struct {
		Value   RawType
		Source  string
		Defined bool
	}{
		Value:   o.Value,
		Source:  o.Source,
		Defined: o.Defined,
	})
}

// String is required for kingpin to generate usage with this datatype
func (o RawTypeOption) String() string {
	if StringifyValue {
		return fmt.Sprintf("%v", o.Value)
	}
	return fmt.Sprintf("{Source:%s Defined:%t Value:%v}", o.Source, o.Defined, o.Value)
}

// MapRawTypeOption is a map of options.
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
func (o MapRawTypeOption) IsCumulative() bool {
	return true
}

// String is required for kingpin to generate usage with this datatype
func (o MapRawTypeOption) String() string {
	return fmt.Sprintf("%v", map[string]RawTypeOption(o))
}

// Map will return a raw map from the Option.
func (o MapRawTypeOption) Map() map[string]RawType {
	tmp := map[string]RawType{}
	for k, v := range o {
		tmp[k] = v.Value
	}
	return tmp
}

// WriteAnswer will assign the option value from the value provided during
// a prompt.  This is useful with survey prompting library
func (o *MapRawTypeOption) WriteAnswer(name string, value interface{}) error {
	tmp := RawTypeOption{}
	if v, ok := value.(RawType); ok {
		tmp.Value = v
		tmp.Defined = true
		tmp.Source = "prompt"
		(*o)[name] = tmp
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", value, tmp.Value, value)
}

// IsDefined will return true if there is more than one key set in the map.
func (o MapRawTypeOption) IsDefined() bool {
	// true if the map has any keys
	return len(o) > 0
}

// ListRawTypeOption is a slice of RawTypeOption
type ListRawTypeOption []RawTypeOption

// Set is required for kingpin interfaces to allow command line params
// to be set to our map datatype
func (o *ListRawTypeOption) Set(value string) error {
	val := RawTypeOption{}
	val.Set(value)
	*o = append(*o, val)
	return nil
}

// WriteAnswer will assign the option value from the value provided during
// a prompt.  This is useful with survey prompting library
func (o *ListRawTypeOption) WriteAnswer(name string, value interface{}) error {
	tmp := RawTypeOption{}
	if v, ok := value.(RawType); ok {
		tmp.Value = v
		tmp.Defined = true
		tmp.Source = "prompt"
		*o = append(*o, tmp)
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", value, tmp.Value, value)
}

// IsCumulative is required for kingpin interfaces to allow multiple values
// to be set on the data structure.
func (o ListRawTypeOption) IsCumulative() bool {
	return true
}

// String is required for kingpin to generate usage with this datatype
func (o ListRawTypeOption) String() string {
	return fmt.Sprintf("%v", []RawTypeOption(o))
}

// Append will add the provided RawType to the slice with NewRawTypeOption
func (o ListRawTypeOption) Append(values ...RawType) ListRawTypeOption {
	results := o
	for _, val := range values {
		results = append(results, NewRawTypeOption(val))
	}
	return results
}

// Slice returns raw []RawType data stored in the ListRawTypeOption
func (o ListRawTypeOption) Slice() []RawType {
	tmp := []RawType{}
	for _, elem := range o {
		tmp = append(tmp, elem.Value)
	}
	return tmp
}

// IsDefined will return true if the ListRawTypeOption has one or more options
// in the slice.
func (o ListRawTypeOption) IsDefined() bool {
	// true if the list is not empty
	return len(o) > 0
}
