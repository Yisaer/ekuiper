package function

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lf-edge/ekuiper/v2/internal/conf"
	"github.com/lf-edge/ekuiper/v2/internal/pkg/def"
	kctx "github.com/lf-edge/ekuiper/v2/internal/topo/context"
	"github.com/lf-edge/ekuiper/v2/internal/topo/state"
)

func TestAccumulateAggCond(t *testing.T) {
	tests := []struct {
		name     string
		results  []interface{}
		testargs [][]interface{}
	}{
		{
			name: "acc_avg",
			testargs: [][]interface{}{
				{int64(1), false, false},
				{int64(1), true, false},
				{int64(1), false, false},
				{int64(1), false, true},
				{int64(1), false, false},
			},
			results: []interface{}{
				float64(0), float64(1), float64(1), float64(1), float64(0),
			},
		},
		{
			name: "acc_min",
			testargs: [][]interface{}{
				{int64(1), false, false},
				{int64(5), true, false},
				{int64(4), false, false},
				{int64(3), false, true},
				{int64(2), false, false},
			},
			results: []interface{}{
				float64(0), float64(5), float64(4), float64(3), float64(0),
			},
		},
		{
			name: "acc_sum",
			testargs: [][]interface{}{
				{int64(1), false, false},
				{int64(1), true, false},
				{int64(1), false, false},
				{int64(1), false, true},
				{int64(1), false, false},
			},
			results: []interface{}{
				float64(0), float64(1), float64(2), float64(3), float64(0),
			},
		},
		{
			name: "acc_count",
			testargs: [][]interface{}{
				{1, false, false},
				{1, true, false},
				{1, false, false},
				{1, false, true},
				{1, false, false},
			},
			results: []interface{}{
				int64(0), int64(1), int64(2), int64(3), int64(0),
			},
		},
		{
			name: "acc_collect",
			testargs: [][]interface{}{
				{int64(1), false, false},
				{int64(1), true, false},
				{int64(2), false, false},
				{int64(3), false, true},
				{int64(4), false, false},
			},
			results: []interface{}{
				[]interface{}{},
				[]interface{}{int64(1)},
				[]interface{}{int64(1), int64(2)},
				[]interface{}{int64(1), int64(2), int64(3)},
				[]interface{}{},
			},
		},
	}
	for _, test := range tests {
		f, ok := builtins[test.name]
		require.True(t, ok)
		contextLogger := conf.Log.WithField("rule", "testExec")
		ctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
		tempStore, _ := state.CreateStore("mockRule0", def.AtMostOnce)
		fctx := kctx.NewDefaultFuncContext(ctx.WithMeta("mockRule0", "test", tempStore), 2)
		for i, arg := range test.testargs {
			newArg := append(arg, true, fmt.Sprintf("%s_key", test.name))
			result, _ := f.exec(fctx, newArg)
			require.Equal(t, test.results[i], result)
		}
	}
}

func TestAccumulateAgg(t *testing.T) {
	tests := []struct {
		name     string
		results  []interface{}
		testargs []interface{}
	}{
		{
			name: "acc_count",
			testargs: []interface{}{
				"1",
				float64(1),
				float32(1),
				1,
				int32(1),
				int64(1),
			},
			results: []interface{}{
				int64(1), int64(2), int64(3), int64(4), int64(5), int64(6),
			},
		},
		{
			name: "acc_avg",
			testargs: []interface{}{
				"1",
				float64(1),
				float64(1),
				float64(1),
				int64(1),
				int64(1),
			},
			results: []interface{}{
				fmt.Errorf("the value should be number"),
				float64(1),
				float64(1),
				float64(1),
				float64(1),
				float64(1),
			},
		},
		{
			name: "acc_max",
			testargs: []interface{}{
				"1",
				float64(1),
				float64(2),
				float64(3),
				int64(4),
				int64(5),
			},
			results: []interface{}{
				fmt.Errorf("the value should be number"),
				float64(1),
				float64(2),
				float64(3),
				float64(4),
				float64(5),
			},
		},
		{
			name: "acc_min",
			testargs: []interface{}{
				"1",
				float64(5),
				float64(4),
				float64(3),
				int64(2),
				int64(1),
			},
			results: []interface{}{
				fmt.Errorf("the value should be number"),
				float64(5),
				float64(4),
				float64(3),
				float64(2),
				float64(1),
			},
		},
		{
			name: "acc_sum",
			testargs: []interface{}{
				"1",
				float64(1),
				float64(1),
				int64(1),
				int64(1),
				int64(1),
			},
			results: []interface{}{
				fmt.Errorf("the value should be number"),
				float64(1),
				float64(2),
				float64(3),
				float64(4),
				float64(5),
			},
		},
		{
			name: "acc_collect",
			testargs: []interface{}{
				int64(1),
				int64(2),
				nil,
				"hello",
				float64(3.14),
			},
			results: []interface{}{
				[]interface{}{int64(1)},
				[]interface{}{int64(1), int64(2)},
				[]interface{}{int64(1), int64(2)},
				[]interface{}{int64(1), int64(2), "hello"},
				[]interface{}{int64(1), int64(2), "hello", float64(3.14)},
			},
		},
	}
	for _, test := range tests {
		f, ok := builtins[test.name]
		require.True(t, ok)
		contextLogger := conf.Log.WithField("rule", "testExec")
		ctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
		tempStore, _ := state.CreateStore("mockRule0", def.AtMostOnce)
		fctx := kctx.NewDefaultFuncContext(ctx.WithMeta("mockRule0", "test", tempStore), 2)
		for i, arg := range test.testargs {
			result, _ := f.exec(fctx, []interface{}{arg, true, fmt.Sprintf("%s_key", test.name)})
			require.Equal(t, test.results[i], result)
		}
	}

	tests2 := []struct {
		name   string
		result interface{}
	}{
		{
			"acc_sum",
			float64(0),
		},
		{
			"acc_max",
			float64(0),
		},
		{
			"acc_min",
			float64(0),
		},
		{
			"acc_avg",
			float64(0),
		},
		{
			"acc_count",
			int64(0),
		},
		{
			"acc_collect",
			[]interface{}{},
		},
	}
	for _, test := range tests2 {
		f, ok := builtins[test.name]
		require.True(t, ok)
		contextLogger := conf.Log.WithField("rule", "testExec")
		ctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
		tempStore, _ := state.CreateStore("mockRule0", def.AtMostOnce)
		fctx := kctx.NewDefaultFuncContext(ctx.WithMeta("mockRule0", "test", tempStore), 2)
		result, b := f.exec(fctx, []interface{}{1, false, fmt.Sprintf("%s_key", test.name)})
		require.True(t, b)
		require.Equal(t, test.result, result)
	}
}

func TestAccMaxBy(t *testing.T) {
	contextLogger := conf.Log.WithField("rule", "testAccMaxBy")
	tctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
	tempStore, _ := state.CreateStore("acc_max_by_test", def.AtMostOnce)
	fctx := kctx.NewDefaultFuncContext(tctx.WithMeta("mockRule0", "test", tempStore), 0)
	f := builtins["acc_max_by"]

	args := func(value, by int64) []interface{} { return []interface{}{value, by, true, "max_by"} }
	result, ok := f.exec(fctx, args(100, 10))
	require.True(t, ok)
	require.Equal(t, int64(100), result)
	result, ok = f.exec(fctx, args(200, 9))
	require.True(t, ok)
	require.Equal(t, int64(100), result)
	result, ok = f.exec(fctx, args(300, 10))
	require.True(t, ok)
	require.Equal(t, int64(300), result)
}

func TestAccMapAgg(t *testing.T) {
	contextLogger := conf.Log.WithField("rule", "testAccMapAgg")
	tctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
	tempStore, _ := state.CreateStore("acc_map_agg_test", def.AtMostOnce)
	fctx := kctx.NewDefaultFuncContext(tctx.WithMeta("mockRule0", "test", tempStore), 0)
	f := builtins["acc_map_agg"]

	args := func(key int64, value interface{}) []interface{} {
		return []interface{}{key, value, true, "map_agg"}
	}
	result, ok := f.exec(fctx, args(18, map[string]interface{}{"max_temp": int64(28)}))
	require.True(t, ok)
	require.Equal(t, []map[string]interface{}{{"key": "18", "value": map[string]interface{}{"max_temp": int64(28)}}}, result)
	_, ok = f.exec(fctx, args(19, map[string]interface{}{"max_temp": int64(31)}))
	require.True(t, ok)
	result, ok = f.exec(fctx, args(18, map[string]interface{}{"max_temp": int64(30)}))
	require.True(t, ok)
	require.Equal(t, []map[string]interface{}{
		{"key": "18", "value": map[string]interface{}{"max_temp": int64(30)}},
		{"key": "19", "value": map[string]interface{}{"max_temp": int64(31)}},
	}, result)
}

func TestAccCollectFuncDirect(t *testing.T) {
	contextLogger := conf.Log.WithField("rule", "testExec")
	ctx := kctx.WithValue(kctx.Background(), kctx.LoggerKey, contextLogger)
	tempStore, _ := state.CreateStore("mockRule0", def.AtMostOnce)
	fctx := kctx.NewDefaultFuncContext(ctx.WithMeta("mockRule0", "test", tempStore), 2)

	cf := accCollectFunc{}

	// Test accReset
	t.Run("accReset", func(t *testing.T) {
		s := &accStatus{Value: int64(42)}
		cf.accReset(s)
		require.Equal(t, []interface{}{}, s.Value)
	})

	// Test accFuncExec with nil Value (defense-in-depth guard)
	t.Run("accFuncExec_nilValue", func(t *testing.T) {
		s := &accStatus{Value: nil}
		cf.accFuncExec(fctx, int64(1), true, "k1", s, true)
		require.Equal(t, []interface{}{int64(1)}, s.Value)
		require.Nil(t, s.Err)
	})

	// Test accFuncExec with nil value arg (should skip)
	t.Run("accFuncExec_nilArg", func(t *testing.T) {
		s := &accStatus{Value: []interface{}{int64(1)}}
		cf.accFuncExec(fctx, nil, true, "k2", s, true)
		require.Equal(t, []interface{}{int64(1)}, s.Value)
	})

	// Test accFuncExec with validData=false and skipStatusSave=true
	t.Run("accFuncExec_invalidData_skipSave", func(t *testing.T) {
		s := &accStatus{Value: []interface{}{int64(1)}}
		cf.accFuncExec(fctx, int64(2), false, "k3", s, true)
		require.Equal(t, []interface{}{int64(1)}, s.Value)
	})

	// Test accFuncExec with skipStatusSave=false (PutState path)
	t.Run("accFuncExec_withSave", func(t *testing.T) {
		s := &accStatus{Value: []interface{}{int64(1)}}
		cf.accFuncExec(fctx, int64(2), true, "k4", s, false)
		require.Equal(t, []interface{}{int64(1), int64(2)}, s.Value)
		require.Nil(t, s.Err)
	})
}
