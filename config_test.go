package config

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
)

var conf *Config

func TestMain(m *testing.M) {
	// Set up a contrived environment
	conf = New("MyAppName")
	conf.TrueStrings = []string{"true", "y"}
	conf.FalseStrings = []string{"false", "no"}
	testfile, err := makeTestFile()
	if err != nil {
		panic(err)
	}
	testdir := filepath.Dir(testfile)
	defer func() {
		derr := os.RemoveAll(testdir)
		if derr != nil {
			panic(derr)
		}
	}()
	conf.FindAndLoad(
		string(os.PathSeparator) + "non_existent_dir",
		os.TempDir() + "non_existent_file.toml",
		testfile,
	)
	if err = os.Setenv("XDG_CONFIG_HOME", testdir); err != nil {
		panic(err)
	}
	if err = os.Setenv("MY_BLANK_ENV_VAR", ""); err != nil {
		panic(err)
	}
	if err = os.Setenv("MY_ENV_VAR", "some bytes"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func makeTestFile() (string, error) {
	tmpdir, err := ioutil.TempDir("", "GoLparConfigTest")
	if err != nil {
		return "", err
	}
	fn := path.Join(tmpdir, "test.toml")
	err = ioutil.WriteFile(fn, []byte(TOML), 0600)
	return fn, err
}

const TOML = `
 alpha = "Some string"
 beta = 42
 gamma = true
 delta = 3.14159
`

func TestConfig_FromFile(t *testing.T) {
	var tests = []struct {
		key    string
		output string
	}{
		{"alpha", "Some string"},
		{"beta", "42"},
		{"gamma", "true"},
		{"delta", "3.14159"},
	}
	for _, tt := range tests {
		r := conf.FromFile(tt.key)
		if r == nil {
			t.Errorf("FromFile(%s) gave nil", tt.key)
		} else {
			if *r != tt.output {
				t.Errorf("FromFile(%s) gave %s, expected %s", tt.key, *r, tt.output)
			}
		}
	}
	if conf.FromFile("zeta") != nil {
		t.Errorf("FromFile(zeta) gave non-nil")
	}
}

func TestConfig_ResolveString(t *testing.T) {
	var tests = []struct {
		input  []*string
		output string
		nerrs  int
	}{
		{[]*string{PS("one")}, "one", 0},
		{[]*string{nil, PS("two")}, "two", 0},
		{[]*string{nil, nil, PS("value with spaces")}, "value with spaces", 0},
		{[]*string{nil, nil, PS("2.71828")}, "2.71828", 0},
		{[]*string{nil, nil}, "", 1},
	}
	lc := New("ResolveString")
	for i, tt := range tests {
		lc.Errors = nil
		r := lc.ResolveString(tt.input...)
		if r != tt.output {
			t.Errorf("ResolveString test %d gave %v, expected %v", i+1, r, tt.output)
		}
		if len(lc.Errors) != tt.nerrs {
			t.Errorf("ResolveString test %d gave %d errors, expected %d", i+1, len(lc.Errors), tt.nerrs)
		}
	}
}

func verify(t *testing.T, funcname string, x *string, y string) {
	if x == nil {
		t.Errorf("%s returned nil, expected %s", funcname, y)
		return
	}
	if *x != y {
		t.Errorf("%s returned %v, expected %v", funcname, *x, y)
	}
}

func TestConfig_toString(t *testing.T) {
	var tests = []struct {
		input  interface{}
		output string
	}{
		{"test value", "test value"},
		{true, "true"},
		{47, "47"},
	}
	for _, tt := range tests {
		out := conf.toString(tt.input)
		if out != tt.output {
			t.Errorf("toString returned %v, expected %v", out, tt.output)
		}
	}
}

func nils(x interface{}) string {
	if x == nil {
		return "nil"
	}
	return "non-nil"
}

func PS(s string) *string {
	return &s
}

func TestConfig_ResolveInt(t *testing.T) {
	var tests = []struct {
		input  []*string
		output int
		nerrs  int
	}{
		{[]*string{PS("1")}, 1, 0},
		{[]*string{nil, PS("0x2f")}, 47, 0},
		{[]*string{nil, nil, PS("")}, 0, 1},
		{[]*string{nil, nil, PS("2.612")}, 2, 0},
		{[]*string{nil, nil, PS("a"), PS("2")}, 2, 1},
	}
	lc := New("ResolveInt")
	for i, tt := range tests {
		lc.Errors = nil
		r := lc.ResolveInt(tt.input...)
		if r != tt.output {
			t.Errorf("ResolveInt test %d gave %v, expected %v", i+1, r, tt.output)
		}
		if len(lc.Errors) != tt.nerrs {
			t.Errorf("ResolveInt test %d gave %d errors, expected %d", i+1, len(lc.Errors), tt.nerrs)
		}
	}
}

func TestConfig_ResolveFloat(t *testing.T) {
	var tests = []struct {
		input  []*string
		output float64
		nerrs  int
	}{
		{[]*string{PS("1")}, 1, 0},
		{[]*string{nil, PS("3.14159")}, 3.14159, 0},
		{[]*string{nil, nil, PS("-1")}, -1.0, 0},
		{[]*string{nil, nil, PS("-2.612")}, -2.612, 0},
		{[]*string{nil, nil, PS("a"), PS("2")}, 2, 1},
	}
	lc := New("ResolveInt")
	for i, tt := range tests {
		lc.Errors = nil
		r := lc.ResolveFloat64(tt.input...)
		if r != tt.output {
			t.Errorf("ResolveInt test %d gave %v, expected %v", i+1, r, tt.output)
		}
		if len(lc.Errors) != tt.nerrs {
			t.Errorf("ResolveInt test %d gave %d errors, expected %d", i+1, len(lc.Errors), tt.nerrs)
		}
	}
}

func TestConfig_stringToBool(t *testing.T) {
	var tests = []struct {
		input  string
		output bool
		ok     bool
	}{
		{"true", true, true},
		{"false", false, true},
		{"y", true, true},
		{"Y", true, true},
		{"no", false, true},
		{"No", false, true},
		{"Maybe", false, false},
	}
	for _, tt := range tests {
		r, ok := conf.stringToBool(tt.input)
		if ok != tt.ok {
			t.Errorf("stringToBool %s gave ok = %v, expected %v", tt.input, ok, tt.ok)
		}
		if r != tt.output {
			t.Errorf("stringToBool %s gave output %v, expected %v", tt.input, r, tt.output)
		}
	}
}

func TestConfig_ResolveBool(t *testing.T) {
	rv1 := conf.ResolveBool(nil, nil, PS("false"), PS("true"))
	if rv1 != false {
		t.Error("conf.ResolveBool test 1 failed")
	}
	rv2 := conf.ResolveBool(nil, nil, PS("true"), PS("false"))
	if rv2 != true {
		t.Error("conf.ResolveBool test 1 failed")
	}
	if !conf.ResolveBool(PS("Y")) {
		t.Errorf("conf.ResolveBool not calling stringToBool properly")
	}
}

func TestConfig_FromEnv(t *testing.T) {
	s := "non-nil"
	nonnil := &s
	var tests = []struct {
		envvar  string
		nilness *string
		value   string
	}{
		{"MY_ENV_VAR", nonnil, "some bytes"},
		{"MY_BLANK_ENV_VAR", nonnil, ""},
		{"MY_UNSET_ENV_VAR", nil, "whatever"},
	}

	for _, tt := range tests {
		ps := conf.FromEnv(tt.envvar)
		if (ps == nil) != (tt.nilness == nil) {
			t.Errorf("FromEnv returned %s %s, expected %s", tt.envvar,
				nils(ps), nils(tt.nilness))
		}
	}
}

func TestHome_UserConfigDir(t *testing.T) {
	c1 := conf.UserHomeDir()
	c2, _ := os.UserHomeDir()
	verify(t, "UserHomeDir", c1, c2)
}

func TestConfig_UserConfigDir(t *testing.T) {
	c1 := conf.UserConfigDir()
	c2, _ := os.UserConfigDir()
	verify(t, "UserConfigDir", c1, c2)
}

func TestConfig_Executable(t *testing.T) {
	c1 := conf.Executable()
	c2, _ := os.Executable()
	verify(t, "Executable", c1, path.Dir(c2))
}

func TestConfig_Default(t *testing.T) {
	testvals := []interface{}{"one value", 2, true}
	retvals := []interface{}{"one value", "2", "true"}
	for i, v := range testvals {
		pt := conf.Default(v)
		if pt == nil {
			t.Errorf("Default gave nil for non-nil %T value %v", v, v)
			return
		}
		if *pt != retvals[i] {
			t.Errorf("Default gave %v for %T %v, expected %v", *pt, v, v, retvals[i])
		}
	}
}
