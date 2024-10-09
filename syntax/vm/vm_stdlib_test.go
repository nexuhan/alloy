package vm_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/grafana/alloy/syntax/alloytypes"
	"github.com/grafana/alloy/syntax/internal/value"
	"github.com/grafana/alloy/syntax/parser"
	"github.com/grafana/alloy/syntax/vm"
	"github.com/stretchr/testify/require"
)

func TestVM_Stdlib(t *testing.T) {
	t.Setenv("TEST_VAR", "Hello!")

	tt := []struct {
		name   string
		input  string
		expect interface{}
	}{
		// deprecated tests
		{"env", `env("TEST_VAR")`, string("Hello!")},
		{"concat", `concat([true, "foo"], [], [false, 1])`, []interface{}{true, "foo", false, 1}},
		{"json_decode object", `json_decode("{\"foo\": \"bar\"}")`, map[string]interface{}{"foo": "bar"}},
		{"yaml_decode object", "yaml_decode(`foo: bar`)", map[string]interface{}{"foo": "bar"}},
		{"base64_decode", `base64_decode("Zm9vYmFyMTIzIT8kKiYoKSctPUB+")`, string(`foobar123!?$*&()'-=@~`)},

		{"sys.env", `sys.env("TEST_VAR")`, string("Hello!")},
		{"array.concat", `array.concat([true, "foo"], [], [false, 1])`, []interface{}{true, "foo", false, 1}},
		{"encoding.from_json object", `encoding.from_json("{\"foo\": \"bar\"}")`, map[string]interface{}{"foo": "bar"}},
		{"encoding.from_json array", `encoding.from_json("[0, 1, 2]")`, []interface{}{float64(0), float64(1), float64(2)}},
		{"encoding.from_json nil field", `encoding.from_json("{\"foo\": null}")`, map[string]interface{}{"foo": nil}},
		{"encoding.from_json nil array element", `encoding.from_json("[0, null]")`, []interface{}{float64(0), nil}},
		{"encoding.from_yaml object", "encoding.from_yaml(`foo: bar`)", map[string]interface{}{"foo": "bar"}},
		{"encoding.from_yaml array", "encoding.from_yaml(`[0, 1, 2]`)", []interface{}{0, 1, 2}},
		{"encoding.from_yaml array float", "encoding.from_yaml(`[0.0, 1.0, 2.0]`)", []interface{}{float64(0), float64(1), float64(2)}},
		{"encoding.from_yaml nil field", "encoding.from_yaml(`foo: null`)", map[string]interface{}{"foo": nil}},
		{"encoding.from_yaml nil array element", `encoding.from_yaml("[0, null]")`, []interface{}{0, nil}},
		{"encoding.from_base64", `encoding.from_base64("Zm9vYmFyMTIzIT8kKiYoKSctPUB+")`, string(`foobar123!?$*&()'-=@~`)},

		// Map tests
		{
			// Basic case. No conflicting key/val pairs.
			"targets.merge",
			`targets.merge([{"a" = "a1", "b" = "b1"}], [{"a" = "a1", "c" = "c1"}], ["a"])`,
			[]map[string]interface{}{{"a": "a1", "b": "b1", "c": "c1"}},
		},
		{
			// The first array has 2 maps, each with the same key/val pairs.
			"targets.merge",
			`targets.merge([{"a" = "a1", "b" = "b1"}, {"a" = "a1", "b" = "b1"}], [{"a" = "a1", "c" = "c1"}], ["a"])`,
			[]map[string]interface{}{{"a": "a1", "b": "b1", "c": "c1"}, {"a": "a1", "b": "b1", "c": "c1"}},
		},
		{
			// Basic case. Integer and string values.
			"targets.merge",
			`targets.merge([{"a" = 1, "b" = 2.2}], [{"a" = 1, "c" = "c1"}], ["a"])`,
			[]map[string]interface{}{{"a": 1, "b": 2.2, "c": "c1"}},
		},
		{
			// The second map will override a value from the first.
			"targets.merge",
			`targets.merge([{"a" = 1, "b" = 2.2}], [{"a" = 1, "b" = "3.3"}], ["a"])`,
			[]map[string]interface{}{{"a": 1, "b": "3.3"}},
		},
		{
			// Not enough matches for a join.
			"targets.merge",
			`targets.merge([{"a" = 1, "b" = 2.2}], [{"a" = 2, "b" = "3.3"}], ["a"])`,
			[]map[string]interface{}{},
		},
		{
			// Not enough matches for a join.
			// The "a" value has differing types.
			"targets.merge",
			`targets.merge([{"a" = 1, "b" = 2.2}], [{"a" = "1", "b" = "3.3"}], ["a"])`,
			[]map[string]interface{}{},
		},
		{
			// Basic case. Some values are arrays and maps.
			"targets.merge",
			`targets.merge([{"a" = 1, "b" = [1,2,3]}], [{"a" = 1, "c" = {"d" = {"e" = 10}}}], ["a"])`,
			[]map[string]interface{}{{"a": 1, "b": []interface{}{1, 2, 3}, "c": map[string]interface{}{"d": map[string]interface{}{"e": 10}}}},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpression(tc.input)
			require.NoError(t, err)

			eval := vm.New(expr)

			rv := reflect.New(reflect.TypeOf(tc.expect))
			require.NoError(t, eval.Evaluate(nil, rv.Interface()))
			require.Equal(t, tc.expect, rv.Elem().Interface())
		})
	}
}

func TestStdlibCoalesce(t *testing.T) {
	t.Setenv("TEST_VAR2", "Hello!")

	tt := []struct {
		name   string
		input  string
		expect interface{}
	}{
		{"coalesce()", `coalesce()`, value.Null},
		{"coalesce(string)", `coalesce("Hello!")`, string("Hello!")},
		{"coalesce(string, string)", `coalesce(sys.env("TEST_VAR2"), "World!")`, string("Hello!")},
		{"(string, string) with fallback", `coalesce(sys.env("NON_DEFINED"), "World!")`, string("World!")},
		{"coalesce(list, list)", `coalesce([], ["fallback"])`, []string{"fallback"}},
		{"coalesce(list, list) with fallback", `coalesce(array.concat(["item"]), ["fallback"])`, []string{"item"}},
		{"coalesce(int, int, int)", `coalesce(0, 1, 2)`, 1},
		{"coalesce(bool, int, int)", `coalesce(false, 1, 2)`, 1},
		{"coalesce(bool, bool)", `coalesce(false, true)`, true},
		{"coalesce(list, bool)", `coalesce(encoding.from_json("[]"), true)`, true},
		{"coalesce(object, true) and return true", `coalesce(encoding.from_json("{}"), true)`, true},
		{"coalesce(object, false) and return false", `coalesce(encoding.from_json("{}"), false)`, false},
		{"coalesce(list, nil)", `coalesce([],null)`, value.Null},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpression(tc.input)
			require.NoError(t, err)

			eval := vm.New(expr)

			rv := reflect.New(reflect.TypeOf(tc.expect))
			require.NoError(t, eval.Evaluate(nil, rv.Interface()))
			require.Equal(t, tc.expect, rv.Elem().Interface())
		})
	}
}

func TestStdlibJsonPath(t *testing.T) {
	tt := []struct {
		name   string
		input  string
		expect interface{}
	}{
		{"json_path with simple json", `json_path("{\"a\": \"b\"}", ".a")`, []string{"b"}},
		{"json_path with simple json without results", `json_path("{\"a\": \"b\"}", ".nonexists")`, []string{}},
		{"json_path with json array", `json_path("[{\"name\": \"Department\",\"value\": \"IT\"},{\"name\":\"ReferenceNumber\",\"value\":\"123456\"},{\"name\":\"TestStatus\",\"value\":\"Pending\"}]", "[?(@.name == \"Department\")].value")`, []string{"IT"}},
		{"json_path with simple json and return first", `json_path("{\"a\": \"b\"}", ".a")[0]`, "b"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpression(tc.input)
			require.NoError(t, err)

			eval := vm.New(expr)

			rv := reflect.New(reflect.TypeOf(tc.expect))
			require.NoError(t, eval.Evaluate(nil, rv.Interface()))
			require.Equal(t, tc.expect, rv.Elem().Interface())
		})
	}
}

func TestStdlib_Nonsensitive(t *testing.T) {
	scope := &vm.Scope{
		Variables: map[string]any{
			"secret":         alloytypes.Secret("foo"),
			"optionalSecret": alloytypes.OptionalSecret{Value: "bar"},
		},
	}

	tt := []struct {
		name   string
		input  string
		expect interface{}
	}{
		// deprecated tests
		{"deprecated secret to string", `nonsensitive(secret)`, string("foo")},

		{"secret to string", `convert.nonsensitive(secret)`, string("foo")},
		{"optional secret to string", `convert.nonsensitive(optionalSecret)`, string("bar")},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpression(tc.input)
			require.NoError(t, err)

			eval := vm.New(expr)

			rv := reflect.New(reflect.TypeOf(tc.expect))
			require.NoError(t, eval.Evaluate(scope, rv.Interface()))
			require.Equal(t, tc.expect, rv.Elem().Interface())
		})
	}
}
func TestStdlib_StringFunc(t *testing.T) {
	scope := &vm.Scope{
		Variables: map[string]any{},
	}

	tt := []struct {
		name   string
		input  string
		expect interface{}
	}{
		// deprecated tests
		{"to_lower", `to_lower("String")`, "string"},
		{"to_upper", `to_upper("string")`, "STRING"},
		{"trimspace", `trim_space("   string \n\n")`, "string"},
		{"trimspace+to_upper+trim", `to_lower(to_upper(trim_space("   String   ")))`, "string"},
		{"split", `split("/aaa/bbb/ccc/ddd", "/")`, []string{"", "aaa", "bbb", "ccc", "ddd"}},
		{"split+index", `split("/aaa/bbb/ccc/ddd", "/")[0]`, ""},
		{"join+split", `join(split("/aaa/bbb/ccc/ddd", "/"), "/")`, "/aaa/bbb/ccc/ddd"},
		{"join", `join(["foo", "bar", "baz"], ", ")`, "foo, bar, baz"},
		{"join w/ int", `join([0, 0, 1], ", ")`, "0, 0, 1"},
		{"format", `format("Hello %s", "World")`, "Hello World"},
		{"format+int", `format("%#v", 1)`, "1"},
		{"format+bool", `format("%#v", true)`, "true"},
		{"format+quote", `format("%q", "hello")`, `"hello"`},
		{"replace", `replace("Hello World", " World", "!")`, "Hello!"},
		{"trim", `trim("?!hello?!", "!?")`, "hello"},
		{"trim2", `trim("   hello! world.!  ", "! ")`, "hello! world."},
		{"trim_prefix", `trim_prefix("helloworld", "hello")`, "world"},
		{"trim_suffix", `trim_suffix("helloworld", "world")`, "hello"},

		{"string.to_lower", `string.to_lower("String")`, "string"},
		{"string.to_upper", `string.to_upper("string")`, "STRING"},
		{"string.trimspace", `string.trim_space("   string \n\n")`, "string"},
		{"string.trimspace+string.to_upper+string.trim", `string.to_lower(string.to_upper(string.trim_space("   String   ")))`, "string"},
		{"string.split", `string.split("/aaa/bbb/ccc/ddd", "/")`, []string{"", "aaa", "bbb", "ccc", "ddd"}},
		{"string.split+index", `string.split("/aaa/bbb/ccc/ddd", "/")[0]`, ""},
		{"string.join+split", `string.join(string.split("/aaa/bbb/ccc/ddd", "/"), "/")`, "/aaa/bbb/ccc/ddd"},
		{"string.join", `string.join(["foo", "bar", "baz"], ", ")`, "foo, bar, baz"},
		{"string.join w/ int", `string.join([0, 0, 1], ", ")`, "0, 0, 1"},
		{"string.format", `string.format("Hello %s", "World")`, "Hello World"},
		{"string.format+int", `string.format("%#v", 1)`, "1"},
		{"string.format+bool", `string.format("%#v", true)`, "true"},
		{"string.format+quote", `string.format("%q", "hello")`, `"hello"`},
		{"string.replace", `string.replace("Hello World", " World", "!")`, "Hello!"},
		{"string.trim", `string.trim("?!hello?!", "!?")`, "hello"},
		{"string.trim2", `string.trim("   hello! world.!  ", "! ")`, "hello! world."},
		{"string.trim_prefix", `string.trim_prefix("helloworld", "hello")`, "world"},
		{"string.trim_suffix", `string.trim_suffix("helloworld", "world")`, "hello"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpression(tc.input)
			require.NoError(t, err)

			eval := vm.New(expr)

			rv := reflect.New(reflect.TypeOf(tc.expect))
			require.NoError(t, eval.Evaluate(scope, rv.Interface()))
			require.Equal(t, tc.expect, rv.Elem().Interface())
		})
	}
}

func TestStdlibFileFunc(t *testing.T) {
	tt := []struct {
		name   string
		input  string
		expect interface{}
	}{
		{"file.path_join", `file.path_join("this/is", "a/path")`, "this/is/a/path"},
		{"file.path_join empty", `file.path_join()`, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpression(tc.input)
			require.NoError(t, err)

			eval := vm.New(expr)

			rv := reflect.New(reflect.TypeOf(tc.expect))
			require.NoError(t, eval.Evaluate(nil, rv.Interface()))
			require.Equal(t, tc.expect, rv.Elem().Interface())
		})
	}
}

func BenchmarkConcat(b *testing.B) {
	// There's a bit of setup work to do here: we want to create a scope holding
	// a slice of the Person type, which has a fair amount of data in it.
	//
	// We then want to pass it through concat.
	//
	// If the code path is fully optimized, there will be no intermediate
	// translations to interface{}.
	type Person struct {
		Name  string            `alloy:"name,attr"`
		Attrs map[string]string `alloy:"attrs,attr"`
	}
	type Body struct {
		Values []Person `alloy:"values,attr"`
	}

	in := `values = array.concat(values_ref)`
	f, err := parser.ParseFile("", []byte(in))
	require.NoError(b, err)

	eval := vm.New(f)

	valuesRef := make([]Person, 0, 20)
	for i := 0; i < 20; i++ {
		data := make(map[string]string, 20)
		for j := 0; j < 20; j++ {
			var (
				key   = fmt.Sprintf("key_%d", i+1)
				value = fmt.Sprintf("value_%d", i+1)
			)
			data[key] = value
		}
		valuesRef = append(valuesRef, Person{
			Name:  "Test Person",
			Attrs: data,
		})
	}
	scope := &vm.Scope{
		Variables: map[string]interface{}{
			"values_ref": valuesRef,
		},
	}

	// Reset timer before running the actual test
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var b Body
		_ = eval.Evaluate(scope, &b)
	}
}
