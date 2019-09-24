package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml"
)

type Basis int

const (
	RelativeToUser Basis = iota // Locate config relative to user's home directory / `XDG_CONFIG_HOME`
	RelativeToExecutable // Locate config relative to the executable, for cloud web applications
)

// Config stores parameters and data needed for loading the configuration from files and the environment.
type Config struct {
	AppName       string // Application name
	FileBase      string // Base name for config file, default "config"
	Location      Basis // Where to locate the config, default RelativeToUser
	fileData      *toml.Tree
	loadAttempted bool // Did we try to lazy-load the config file yet?
	PrefsFile     string // The resolved name of the preferences file, for display in your error messages
	Errors        []error // List of errors encountered while trying to load the config
	Warnings      []error // List of warnings encountered while trying to load the config
	TrueStrings   []string // String values which count as `true` (case-insensitive), default `["true"]`
	FalseStrings  []string // String values which count as `false` (case-insensitive), default `["false"]`
}

// New returns a Config object which can be used to look up configuration values from the environment
// and from a TOML file.
func New(appname string) *Config {
	return &Config{
		AppName:      appname,
		FileBase:     "config",
		Location:     RelativeToUser,
		TrueStrings:  []string{"true"},
		FalseStrings: []string{"false"},
	}
}

// prefsFileName works out the appropriate preferences file name
func (c *Config) prefsFileName() (string, error) {
	if c.Location == RelativeToExecutable {
		dir, err := os.Executable()
		if err != nil {
			return dir, err
		}
		return filepath.Join(dir, c.FileBase+".toml"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return dir, err
	}
	return filepath.Join(dir, c.AppName, c.FileBase+".toml"), nil
}

// tomlError adds a TOML-related error to the list of errors
func (c *Config) tomlError(filename string, stage string, err error) {
	werr := fmt.Errorf("can't %s TOML preference file %s: %w", stage, filename, err)
	if stage == "locate" {
		c.Warnings = append(c.Warnings, werr)
	} else {
		c.Errors = append(c.Errors, werr)
	}
}

// loadTOML attempts to lazy-load the TOML file, if an attempt hasn't already been made.
func (c *Config) loadTOML() {
	if c.loadAttempted {
		return
	}
	c.loadAttempted = true
	pfile, err := c.prefsFileName()
	if err != nil {
		c.Errors = append(c.Errors, err)
		return
	}
	pf, err := os.Open(pfile)
	if err != nil {
		c.Warnings = append(c.Warnings, err)
		return
	}
	defer func () {
		err := pf.Close()
		if err != nil {
			c.Errors = append(c.Errors, err)
		}
	}()
	filedata, err := toml.LoadReader(pf)
	if err != nil {
		c.Errors = append(c.Errors, err)
		return
	}
	c.fileData = filedata
	c.PrefsFile = pfile
}

// FromFile obtains a configuration value from the TOML config file, given a string key.
func (c *Config) FromFile(key string) *string {
	c.loadTOML()
	if c.fileData == nil {
		return nil
	}
	if c.fileData.Has(key) {
		v := c.fileData.Get(key)
		x := c.toString(v)
		return &x
	}
	return nil
}

// ResolveString loops through the listed possible values to find a non-missing one,
// and return it. If no values are present, you get the zero string `""`.
func (c *Config) ResolveString(list ...*string) string {
	for _, elem := range list {
		if elem != nil {
			return *elem
		}
	}
	c.Errors = append(c.Errors, fmt.Errorf("missing default string value"))
	return ""
}

// toString converts an int, bool or string to a string; anything else ends up as empty string
func (c *Config) toString(x interface{}) string {
	switch v := x.(type) {
	case int64:
		return strconv.FormatInt(v,10)
	case int:
		return strconv.FormatInt(int64(v),10)
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v,'f',-1,64)
	case string:
		return v
	}
	c.Errors = append(c.Errors, fmt.Errorf("unexpected data type %T", x))
	return ""
}

// ResolveInt loops through the listed possible values to find a non-missing one,
// then parses it and casts it to an integer. If no values are present,
// you get the zero integer value `0`.
func (c *Config) ResolveInt(list ...*string) int {
	for _, elem := range list {
		if elem != nil && *elem != "" {
			var val int64
			var err error
			if strings.Contains(*elem, ".") {
				var v float32
				var tv float64
				tv, err = strconv.ParseFloat(*elem, 32)
				v = float32(tv)
				val = int64(v)
			} else {
				val, err = strconv.ParseInt(*elem, 0, 64)
			}
			if err != nil {
				c.Errors = append(c.Errors, fmt.Errorf("unrecognized numeric value '%s': %w", *elem, err))
			} else {
				return int(val)
			}
		}
	}
	c.Errors = append(c.Errors, fmt.Errorf("missing default int value"))
	return 0
}

// stringToBool interprets a string as a bool, given the lists of TrueStrings and FalseStrings.
// If there's a match, you get the decoded boolean, and ok = true.
// Values which don't match either list result in ok = false.
func (c *Config) stringToBool(bs string) (value bool, ok bool) {
	tbs := strings.TrimSpace(bs)
	for _, ts := range c.TrueStrings {
		if strings.EqualFold(tbs, ts) {
			return true, true
		}
	}
	for _, ts := range c.FalseStrings {
		if strings.EqualFold(tbs, ts) {
			return false, true
		}
	}
	return false, false
}

// ResolveBool loops through the listed possible values to find a non-missing one,
// then parses it and casts it to a boolean. If no values are present,
// you get the zero boolean value `false`.
func (c *Config) ResolveBool(list ...*string) bool {
	for _, elem := range list {
		if elem != nil && *elem != "" {
			b, ok := c.stringToBool(*elem)
			if !ok {
				c.Errors = append(c.Errors, fmt.Errorf("unrecognized bool value %s", *elem))
			}
			return b
		}
	}
	c.Errors = append(c.Errors, fmt.Errorf("missing default bool value"))
	return false
}

// FromEnv looks for a value in an environment variable with the specified name.
func (c *Config) FromEnv(key string) *string {
	x, ok := os.LookupEnv(key)
	if ok {
		return &x
	}
	return nil
}

// UserHomeDir is a wrapped version of os.UserHomeDir which appends any error to Config.Errors.
func (c *Config) UserHomeDir() *string {
	home, err := os.UserHomeDir()
	if err != nil {
		c.Errors = append(c.Errors, fmt.Errorf("couldn't locate home directory: %w", err))
		return nil
	}
	return &home
}

// UserConfigDir is a wrapped version of os.UserConfigDir which appends any error to Config.Errors.
func (c *Config) UserConfigDir() *string {
	home, err := os.UserConfigDir()
	if err != nil {
		c.Errors = append(c.Errors, fmt.Errorf("couldn't locate home directory: %w", err))
		return nil
	}
	return &home
}

// Executable is a wrapped version of os.Executable which appends any error to Config.Errors.
func (c *Config) Executable() *string {
	exe, err := os.Executable()
	if err != nil {
		c.Errors = append(c.Errors, fmt.Errorf("couldn't locate executable: %w", err))
		return nil
	}
	d := path.Dir(exe)
	return &d
}

// Default wraps an int, bool or string value to act as default when resolving config values.
func (c *Config) Default(x interface{}) *string {
	if x != nil {
		switch y := x.(type) {
		case bool:
			s := strconv.FormatBool(y)
			return &s
		case int:
			s := strconv.Itoa(y)
			return &s
		case string:
			return &y
		default:
			return nil
		}
	}
	return nil
}

