package figtree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionInterface(t *testing.T) {
	f := func(_ Option) bool {
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
