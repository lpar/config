package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/lpar/config"
)

func main() {
	loadConfig()
}

// An example of what your config-loading method might look like.
func loadConfig() {

	// Create a config object, supplying our application name.
	conf := config.New("MyAppName")

	// Add some extra allowed values for `true` and `false`.
	conf.TrueStrings = []string{"true", "yes"}
	conf.FalseStrings = []string{"false", "no"}

	// Load a config file from one of the following places, in order of preference
	locn := conf.FindAndLoad(
		conf.FileFromExecutable(),
		conf.FileFromHome(),
		"/tmp/test.toml",
	)

	if locn != "" {
		fmt.Printf("Loaded config from %s\n", locn)
	}

	// Define a boolean debug flag with the following resolution rules:
	// - If the user supplies `-debug` on the command line, use that.
	// - If not, look for a `DEBUG` environment variable.
	// - If that's not found, look for a `debug = true|false` setting in the config file.
	// - If that's not found, the default is debug = false.
	debug := flag.Bool("debug",
		conf.ResolveBool(
			conf.FromEnv("DEBUG"),
			conf.FromFile("debug"),
			conf.Default(false),
		), "Whether to run in debug mode")

	// Define a string base directory value with the following resolution rules:
	// - If the user supplies a `-baseDir` on the command line, use that.
	// - Otherwise, first check `APP_DIR`, then the `home = "/path/to/home"` line in the config file.
	// - If none of those sources were found, check the user's OS home directory as per Go's `os.UserHomeDir()`.
	// - If that failed for some reason, use the `HOME` environment variable.
	// - If even that failed, use a hard-coded value.
	base := flag.String("baseDir",
		conf.ResolveString(
			conf.FromEnv("APP_DIR"),
			conf.FromFile("home"),
			conf.UserHomeDir(),
			conf.Executable(),
			conf.FromEnv("HOME"),
			conf.Default("/home/meta"),
		), "Base directory for application")

	// Define an age integer variable with the following rules:
	// - If `-age` is on the command line, use the corresponding value.
	// - Otherwise, look for `USER_AGE` in the environment.
	// - Otherwise, check the config file.
	// - Otherwise, assume it's 16.
	age := flag.Int("age",
		conf.ResolveInt(
			conf.FromEnv("USER_AGE"),
			conf.FromFile("age"),
			conf.Default(16),
		),
		"Age of user")

	// Standard call to `flag.Parse()` activates and populates all the variables declared above.
	flag.Parse()

	// Run through and print any errors. Normally you'd likely want to stop if there's a non-zero number of errors,
	// but that's up to you.
	for _, err := range conf.Errors {
		log.Printf("error: %v", err)
	}

	// In a real application we'd maybe return the resolved config values here, but for this example we'll just
	// output them.
	log.Printf("debug = %v", *debug)
	log.Printf("baseDir = %v", *base)
	log.Printf("age = %v", *age)
}
