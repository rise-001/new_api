package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCustomMenuItemsJSON(t *testing.T) {
	valid := `[
		{"id":"docs","url":"https://docs.example.com","name":"Docs","location":"top","open_mode":"new_tab"},
		{"id":"support","url":"https://support.example.com/chat","name":"Support","location":"chat","open_mode":"embed"},
		{"id":"account","url":"https://account.example.com","name":"Account","location":"personal","open_mode":"embed"}
	]`
	require.NoError(t, ValidateCustomMenuItemsJSON(valid))

	tests := []struct {
		name  string
		value string
	}{
		{name: "rejects non-array JSON", value: `{}`},
		{name: "rejects duplicate ids", value: `[{"id":"same","url":"https://one.example.com","name":"One","location":"chat","open_mode":"embed"},{"id":"same","url":"https://two.example.com","name":"Two","location":"chat","open_mode":"new_tab"}]`},
		{name: "rejects unsafe URL schemes", value: `[{"id":"unsafe","url":"javascript:alert(1)","name":"Unsafe","location":"chat","open_mode":"embed"}]`},
		{name: "rejects embedded top navigation", value: `[{"id":"docs","url":"https://docs.example.com","name":"Docs","location":"top","open_mode":"embed"}]`},
		{name: "rejects invalid chat open mode", value: `[{"id":"chat","url":"https://chat.example.com","name":"Chat","location":"chat","open_mode":"same_tab"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, ValidateCustomMenuItemsJSON(tt.value))
		})
	}
}
