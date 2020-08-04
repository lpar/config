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

// Basis is an enum used for indicating the basis for locating the config file.
type Basis int

// Config stores parameters and data needed for loading the configuration from files and the environment.
type Config struct {
	AppName      string // Application name
	FileBase     string // Base name for config file, default "config"
	Location     Basis  // Where to locate the config, default ORelativeToUser
	fileData     *toml.Tree
	Errors       []error  // List of errors encountered while trying to load the config
	TrueStrings  []string // String values which count as `true` (case-insensitive), default `["true"]`
	FalseStrings []string // String values which count as `false` (case-insensitive), default `["false"]`
}

// New returns a Config object which can be used to look up configuration values from the environment
// and from a TOML file.
func New(appname string) *Config {
	return &Config{
		AppName:      appname,
		FileBase:     "config",
		TrueStrings:  []string{"true"},
		FalseStrings: []string{"false"},
	}
}

// --- File resolving ---

// FileFromExecutable computes the config file name based on the location of executable.
// Used for cloud applications.
func (c *Config) FileFromExecutable() string {
	dir, err := os.Executable()
	if err != nil {
		c.Errors = append(c.Errors, err)
		return ""
	}
	return filepath.Join(filepath.Dir(dir), c.FileBase+".toml")
}

// FileFromHome looks for the config file in the standard location for the user's OS, as per Go's
// `os.UserConfigDir`. Example default filenames:
//  Linux: ~/.config/AppName/config.toml
//  Mac: ~/Library/Application Support/AppName/config.toml
//  Windows: %AppData%\AppName\config.toml
func (c *Config) FileFromHome() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		c.Errors = append(c.Errors, err)
		return ""
	}
	return filepath.Join(dir, c.AppName, c.FileBase+".toml")
}

func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Find locates the first extant TOML file by checking the supplied list of possible locations.
// Empty strings are ignored. It returns the filename.
func (c *Config) Find(list ...string) string {
	for _, elem := range list {
		if elem != "" {
			exists, err := fileExists(elem)
			if err != nil {
				fmt.Printf("%v", err)
				continue
			}
			if exists {
				return elem
			}
		}
	}
	return ""
}

// Load loads the TOML config file specified. Any errors are appended to Config.Errors
func (c *Config) Load(filename string) {
	pf, err := os.Open(filename)
	if err != nil {
		c.Errors = append(c.Errors, err)
		return
	}
	defer func() {
		err = pf.Close()
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
}

// FindAndLoad locates the first config file from the list of possibilities, then loads it.
// Empty strings are ignored, and the name of the file that was loaded is returned.
// It's equivalent to Find followed by Load.
func (c *Config) FindAndLoad(list ...string) string {
	fn := c.Find(list...)
	if fn != "" {
		c.Load(fn)
	}
	return fn
}

// --- value resolution

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
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case bool:
		return strconv.FormatBool(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	}
	c.Errors = append(c.Errors, fmt.Errorf("unexpected data type %T", x))
	return ""
}

// ResolveInt loops through the listed possible values to find a non-missing one,
// then parses it and casts it to an integer. If no values are present,
// you get the zero integer value `0`. Floating point values are rounded down.
func (c *Config) ResolveInt(list ...*string) int {
	for _, elem := range list {
		if elem != nil && *elem != "" {
			var val int64
			var err error
			if strings.Contains(*elem, ".") {
				var v float32
				var tv float64
				tv, err = strconv.ParseFloat(*elem, 64)
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

// ResolveFloat64 loops through the listed possible values to find a non-missing one,
// then parses it and casts it to a float64. If no values are present,
// you get the zero value.
func (c *Config) ResolveFloat64(list ...*string) float64 {
	for _, elem := range list {
		if elem != nil && *elem != "" {
			val, err := strconv.ParseFloat(*elem, 64)
			if err != nil {
				c.Errors = append(c.Errors, fmt.Errorf("unrecognized numeric value '%s': %w", *elem, err))
			} else {
				return val
			}
		}
	}
	c.Errors = append(c.Errors, fmt.Errorf("missing default int value"))
	return 0.0
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

// FromFile obtains a configuration value from the TOML config file, given a string key.
func (c *Config) FromFile(key string) *string {
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
