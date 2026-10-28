package application

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rpcFixture struct{}

func (rpcFixture) Shout(s string) (string, error) { return s + "!", nil }
func (rpcFixture) Fails() (string, error)         { return "", errors.New("boom") }
func (rpcFixture) OnlyError() error               { return nil }

func TestCallMethod(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		args    string
		want    any
		wantErr string
	}{
		{name: "returns value", method: "Shout", args: `["hi"]`, want: "hi!"},
		{name: "void method returns nil", method: "OnlyError", args: `[]`},
		{name: "nil error is not returned as value", method: "Shout", args: `["a"]`, want: "a!"},
		{name: "error is propagated", method: "Fails", args: `[]`, wantErr: "boom"},
		{name: "unknown method", method: "Nope", args: `[]`, wantErr: `unknown method "Nope"`},
		{name: "wrong arg count", method: "Shout", args: `[]`, wantErr: `method "Shout" takes 1 arguments, got 0`},
		{name: "invalid arg", method: "Shout", args: `[1]`, wantErr: "argument 1 of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(tt.args), &args))

			got, err := callMethod(rpcFixture{}, tt.method, args)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
