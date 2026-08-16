package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfo_String(t *testing.T) {
	i := Info{Version: "1.0.0", BuildHash: "abc1234", BuildDate: "2026-08-11T10:00:00Z"}
	assert.Equal(t, "1.0.0 (abc1234, 2026-08-11T10:00:00Z)", i.String())
}

func TestInfo_String_EmptyFields(t *testing.T) {
	i := Info{}
	assert.Equal(t, " (, )", i.String())
}
