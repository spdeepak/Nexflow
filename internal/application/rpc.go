package application

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// HandleRPC invokes an exported App method by name with JSON arguments.
// It lets the frontend run in a plain browser, where Wails injects no bindings.
func (a *App) HandleRPC(name string, args []json.RawMessage) (any, error) {
	return callMethod(a, name, args)
}

// callMethod mirrors a Wails binding call: JSON args in, first non-error
// return value out.
func callMethod(target any, name string, args []json.RawMessage) (any, error) {
	method := reflect.ValueOf(target).MethodByName(name)
	if !method.IsValid() {
		return nil, fmt.Errorf("unknown method %q", name)
	}

	typ := method.Type()
	if len(args) != typ.NumIn() {
		return nil, fmt.Errorf("method %q takes %d arguments, got %d", name, typ.NumIn(), len(args))
	}

	in := make([]reflect.Value, len(args))
	for i, raw := range args {
		arg := reflect.New(typ.In(i))
		if err := json.Unmarshal(raw, arg.Interface()); err != nil {
			return nil, fmt.Errorf("argument %d of %q: %w", i+1, name, err)
		}
		in[i] = arg.Elem()
	}

	out := method.Call(in)

	errorType := reflect.TypeFor[error]()
	if n := len(out); n > 0 && typ.Out(n-1).Implements(errorType) {
		if errVal := out[n-1]; !errVal.IsNil() {
			return nil, errVal.Interface().(error)
		}
		out = out[:n-1]
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out[0].Interface(), nil
}
