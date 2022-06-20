package figtree

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"gopkg.in/yaml.v3"
)

type option interface {
	IsDefined() bool
	GetValue() any
	SetValue(any) error
	SetSource(string)
	GetSource() string
}

// StringifyValue is global variable to indicate if the Option should be
// serialized as just the value (when value is true) or if the entire Option
// struct should be serialized.  This is a hack, and not recommended for general
// usage, but can be useful for debugging.
var StringifyValue = true

// stringMapRegex is used in option parsing for map types Set routines
var stringMapRegex = regexp.MustCompile("[:=]")

type Option[T any] struct {
	Source  string
	Defined bool
	Value   T
}

func NewOption[T any](dflt T) Option[T] {
	return Option[T]{
		Source:  "default",
		Defined: true,
		Value:   dflt,
	}
}

func (o Option[T]) IsDefined() bool {
	return o.Defined
}

func (o *Option[T]) SetSource(source string) {
	o.Source = source
}

func (o *Option[T]) GetSource() string {
	return o.Source
}

func (o Option[T]) GetValue() any {
	return o.Value
}

// WriteAnswer implements the Settable interface as defined by the
// survey prompting library:
// https://github.com/AlecAivazis/survey/blob/v2.3.5/core/write.go#L15-L18
func (o *Option[T]) WriteAnswer(name string, value any) error {
	if v, ok := value.(T); ok {
		o.Value = v
		o.Defined = true
		o.Source = "prompt"
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", value, o.Value, value)
}

// Set implements part of the Value interface as defined by the kingpin command
// line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L26-L29
func (o *Option[T]) Set(s string) error {
	err := convertString(s, &o.Value)
	if err != nil {
		return err
	}
	o.Source = "override"
	o.Defined = true
	return nil
}

// String implements part of the Value interface as defined by the kingpin
// command line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L26-L29
func (o Option[T]) String() string {
	if StringifyValue {
		return fmt.Sprint(o.Value)
	}
	return fmt.Sprintf("{Source:%s Defined:%t Value:%v}", o.Source, o.Defined, o.Value)
}

// SetValue implements the Settings interface as defined by the kingpin
// command line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/parsers.go#L13-L15
func (o *Option[T]) SetValue(v any) error {
	if val, ok := v.(T); ok {
		o.Value = val
		o.Defined = true
		return nil
	}
	panic(fmt.Sprintf("Got %T expected %T type: %v", v, o.Value, v))
}

// UnmarshalYAML implement the Unmarshaler interface used by the
// yaml library:
// https://github.com/go-yaml/yaml/blob/v3.0.1/yaml.go#L36-L38
func (o *Option[T]) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&o.Value); err != nil {
		return err
	}
	o.Source = "yaml"
	o.Defined = true
	return nil
}

// MarshalYAML implements the Marshaler interface used by the yaml library:
// https://github.com/go-yaml/yaml/blob/v3.0.1/yaml.go#L50-L52
func (o Option[T]) MarshalYAML() (any, error) {
	if StringifyValue {
		return o.Value, nil
	}
	// need a copy of this struct without the MarshalYAML interface attached
	return struct {
		Value   T
		Source  string
		Defined bool
	}{
		Value:   o.Value,
		Source:  o.Source,
		Defined: o.Defined,
	}, nil
}

// UnmarshalJSON implements the Unmarshaler interface as defined by json:
// https://cs.opensource.google/go/go/+/refs/tags/go1.18.3:src/encoding/json/decode.go;l=118-120
func (o *Option[T]) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &o.Value); err != nil {
		return err
	}
	o.Source = "json"
	o.Defined = true
	return nil
}

// MarshalJSON implements the Marshaler interface as defined by json:
// https://cs.opensource.google/go/go/+/refs/tags/go1.18.3:src/encoding/json/encode.go;l=225-227
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if StringifyValue {
		return json.Marshal(o.Value)
	}
	// need a copy of this struct without the MarshalJSON interface attached
	return json.Marshal(struct {
		Value   T
		Source  string
		Defined bool
	}{
		Value:   o.Value,
		Source:  o.Source,
		Defined: o.Defined,
	})
}

// IsBoolFlag implements part of the boolFlag interface as defined by the
// kingpin command line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L42-L45
func (o Option[T]) IsBoolFlag() bool {
	// TODO hopefully Go will get template specializations so we can
	// implement this function specifically for Option[bool], but for
	// now we have to use runtime reflection to determine the type.
	v := reflect.ValueOf(o.Value)
	if v.Kind() == reflect.Bool {
		return true
	}
	return false
}

type MapOption[T any] map[string]Option[T]

// Set implements part of the Value interface as defined by the kingpin command
// line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L26-L29
func (o *MapOption[T]) Set(value string) error {
	parts := stringMapRegex.Split(value, 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected KEY=VALUE got '%s'", value)
	}
	val := Option[T]{}
	val.Set(parts[1])
	(*o)[parts[0]] = val
	return nil
}

// IsCumulative implements part of the remainderArg interface as defined by the
// kingpin command line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L49-L52
func (o MapOption[T]) IsCumulative() bool {
	return true
}

// String implements part of the Value interface as defined by the kingpin
// command line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L26-L29
func (o MapOption[T]) String() string {
	return fmt.Sprint(map[string]Option[T](o))
}

func (o MapOption[T]) Map() map[string]T {
	tmp := map[string]T{}
	for k, v := range o {
		tmp[k] = v.Value
	}
	return tmp
}

// WriteAnswer implements the Settable interface as defined by the
// survey prompting library:
// https://github.com/AlecAivazis/survey/blob/v2.3.5/core/write.go#L15-L18
func (o *MapOption[T]) WriteAnswer(name string, value any) error {
	tmp := Option[T]{}
	if v, ok := value.(T); ok {
		tmp.Value = v
		tmp.Defined = true
		tmp.Source = "prompt"
		(*o)[name] = tmp
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", value, tmp.Value, value)
}

func (o MapOption[T]) IsDefined() bool {
	// true if the map has any keys
	return len(o) > 0
}

type ListOption[T any] []Option[T]

// Set implements part of the Value interface as defined by the kingpin command
// line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L26-L29
func (o *ListOption[T]) Set(value string) error {
	val := Option[T]{}
	val.Set(value)
	*o = append(*o, val)
	return nil
}

// WriteAnswer implements the Settable interface as defined by the
// survey prompting library:
// https://github.com/AlecAivazis/survey/blob/v2.3.5/core/write.go#L15-L18
func (o *ListOption[T]) WriteAnswer(name string, value any) error {
	tmp := Option[T]{}
	if v, ok := value.(T); ok {
		tmp.Value = v
		tmp.Defined = true
		tmp.Source = "prompt"
		*o = append(*o, tmp)
		return nil
	}
	return fmt.Errorf("Got %T expected %T type: %v", value, tmp.Value, value)
}

// IsCumulative implements part of the remainderArg interface as defined by the
// kingpin command line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L49-L52
func (o ListOption[T]) IsCumulative() bool {
	return true
}

// String implements part of the Value interface as defined by the kingpin
// command line option library:
// https://github.com/alecthomas/kingpin/blob/v1.3.4/values.go#L26-L29
func (o ListOption[T]) String() string {
	return fmt.Sprint([]Option[T](o))
}

func (o ListOption[T]) Append(values ...T) ListOption[T] {
	results := o
	for _, val := range values {
		results = append(results, NewOption[T](val))
	}
	return results
}

func (o ListOption[T]) Slice() []T {
	tmp := []T{}
	for _, elem := range o {
		tmp = append(tmp, elem.Value)
	}
	return tmp
}

func (o ListOption[T]) IsDefined() bool {
	// true if the list is not empty
	return len(o) > 0
}
