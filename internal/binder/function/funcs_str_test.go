// Copyright 2023 EMQ Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package function

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lf-edge/ekuiper/internal/conf"
	kctx "github.com/lf-edge/ekuiper/internal/topo/context"
	"github.com/lf-edge/ekuiper/internal/topo/state"
	"github.com/lf-edge/ekuiper/pkg/api"
)

func TestStrFuncNil(t *testing.T) {
	contextLogger := conf.Log.WithField("rule", "testExec")
	ctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
	tempStore, _ := state.CreateStore("mockRule0", api.AtMostOnce)
	fctx := kctx.NewDefaultFuncContext(ctx.WithMeta("mockRule0", "test", tempStore), 2)
	oldBuiltins := builtins
	defer func() {
		builtins = oldBuiltins
	}()
	builtins = map[string]builtinFunc{}
	registerStrFunc()
	for name, function := range builtins {
		switch name {
		case "concat":
			r, b := function.exec(fctx, []interface{}{"1", nil, "2"})
			require.True(t, b, fmt.Sprintf("%v failed", name))
			require.Equal(t, "12", r)
		case "endswith", "regexp_matches", "startswith":
			r, b := function.check([]interface{}{nil})
			require.True(t, b, fmt.Sprintf("%v failed", name))
			require.Equal(t, false, r)
		case "indexof":
			r, b := function.exec(fctx, []interface{}{nil})
			require.True(t, b, fmt.Sprintf("%v failed", name))
			require.Equal(t, -1, r)
		case "length", "numbytes":
			r, b := function.check([]interface{}{nil})
			require.True(t, b, fmt.Sprintf("%v failed", name))
			require.Equal(t, 0, r)
		default:
			r, b := function.check([]interface{}{nil})
			require.True(t, b, fmt.Sprintf("%v failed", name))
			require.Nil(t, r, fmt.Sprintf("%v failed", name))
		}
	}
}

func TestSplitValueFunctions(t *testing.T) {
	f, ok := builtins["split_value"]
	if !ok {
		t.Fatal("builtin not found")
	}
	contextLogger := conf.Log.WithField("rule", "testExec")
	ctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
	tempStore, _ := state.CreateStore("mockRule0", api.AtMostOnce)
	fctx := kctx.NewDefaultFuncContext(ctx.WithMeta("mockRule0", "test", tempStore), 2)
	tests := []struct {
		args   []interface{}
		result interface{}
		ok     bool
	}{
		{ // 0
			args:   []interface{}{"a/b/c", "/", 0},
			result: "a",
			ok:     true,
		},
		{ // 0
			args:   []interface{}{"a/b/c", "/", -1},
			result: "c",
			ok:     true,
		},
		{ // 0
			args:   []interface{}{"a/b/c", "/", 3},
			result: errors.New("3 out of index array (size = 3)"),
			ok:     false,
		},
		{ // 0
			args:   []interface{}{"a/b/c", "/", -4},
			result: errors.New("-4 out of index array (size = 3)"),
			ok:     false,
		},
	}
	for _, tt := range tests {
		result, ok := f.exec(fctx, tt.args)
		require.Equal(t, tt.ok, ok)
		require.Equal(t, tt.result, result)
	}
}
