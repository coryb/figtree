package figtree

type Option interface {
	IsDefined() bool
	GetValue() interface{}
	SetValue(interface{}) error
	SetSource(string)
}

var StringifyValue = true
