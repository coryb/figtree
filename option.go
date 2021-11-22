package figtree

import "regexp"

type option interface {
	IsDefined() bool
	GetValue() interface{}
	SetValue(interface{}) error
	SetSource(string)
	GetSource() string
}

// StringifyValue is a global value that can be use for debugging to control
// how options are unmarshalled.  If true, then just the value will be
// unmarshalled, otherwise the Source, Value and Defined fields will be
// unmarshalled as a single struct.
var StringifyValue = true

// used in option parsing for map types Set routines
var stringMapRegex = regexp.MustCompile("[:=]")

// IsBoolFlag is required by kingpin interface to determine if
// this variable requires a value
func (b BoolOption) IsBoolFlag() bool {
	return true
}
