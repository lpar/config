
# lpar/config

A library which helps you load and resolve basic configuration from command line flags, environment variables and a TOML config file. You can think of it as a minimal compact construction kit for building your own `LoadConfig` method to call from `main`.

## Features

Scroll down for longer rationale, but here are the feature highlights:

 - Under 250 lines of code.
 - Only one external dependency (for the TOML handling).
 - Has you leverage the standard `flag` package for command line flags, also works fine with drop-in replacements like `pflag`.
 - Obeys `XDG_CONFIG_DIR` for regular applications, or works in cloud app mode to store config next to the executable.
 - No need to construct structs or annotate them as it doesn't use struct reflection.
 - Define your own prioritization rules for environment variables, command line flags and file data.
 - Add your own additional acceptable values for `true` and `false` (like `yes`, `no`).
 - No singletons. Have multiple different sets of config rules if you want.

And here are some key limitations:

 - Only supports TOML for the config file format, for now. (See discussion below.)
 - Doesn't write config files, only reads them.
 - Because command line arguments and environment variables are [stringly typed][st], for consistency TOML configuration information is handled in a non-type-enforcing way. For example, you can supply numbers as quoted strings _or_ bare numbers in your TOML file. I guess that might also be a feature to some people, though.
 - It's not easy to adjust how command line flags are interpreted based on the config file, or change the config file name based on command line flags, because of how the `flags` package works. (I'd be interested to hear ideas for how to solve that problem, it might be possible to parse command line flags in multiple passes using `flags` and I just haven't worked out how yet?)

[st]: https://www.techopedia.com/definition/31876/stringly-typed
[pflag]: https://github.com/spf13/pflag

## Usage example

See `[example/example.go][example]` for an excessively commented example.

[example]: https://github.com/lpar/config/blob/master/example/example.go

## License

Same license as Go, see LICENSE file.
 
## Why did you write this?

Yes, I know, there are [lots of configuration libraries out there][libs]. However, I didn't like any of them, they all seemed 
to suffer from one of the following problems:

 1. Lack of flexibility. For example, [ff][ff] always expects config file values to override environment values, and [koan][koan] always does the reverse. I sometimes want environment to override the config file (e.g. detecting Cloud Foundry), and sometimes want the config file to override the environment (e.g. finding the HOME directory).
 2. Complexity. I've used [viper][viper], but it's a bit imposing. Five different methods just for implementing reading environment variables, for example.
 3. Bloat. There's [gookit/config][goo], which looks simple enough to use, but it's 2,600 lines of code with another 26,920 lines of code in dependencies.
 4. UDOG/YAGNI issues. Some of the libraries seem to suffer from an Unnecessary Degree Of Generality, offering to let me define my own flag provider to support any custom flag syntax or file format. Others provide facilities to merge multiple config files and validate them against a schema. I just want to read a simple config so my app can start. I don't need a built-in `etcd` or Consul client, chances are You Ain't Gonna Need That.
 
[libs]: https://github.com/avelino/awesome-go#configuration 
[ff]: https://github.com/peterbourgon/ff
[koan]: https://github.com/knadh/koanf
[viper]: https://github.com/spf13/viper
[goo]: https://github.com/gookit/config

## OK, but why TOML?

I'm not wild about TOML. However, [YAML is awful][yaml], JSON doesn't allow comments, and XML is annoying to edit with a text editor. I like the look of [HJSON][hjson], [JSON5][json5], [HCL][hcl] and [HOCON][hocon], in that they all provide JSON-but-with-comments, but I don't like that there are four different improved JSON variants out there; it makes me want to steer clear of all of them. TOML does the job. So I picked a TOML library for Go that doesn't require reflection and seems to be actively maintained.

[yaml]: https://noyaml.com/
[hjson]: https://hjson.org/
[json5]: https://json5.org/
[hcl]: https://github.com/hashicorp/hcl
[hocon]: https://github.com/lightbend/config/blob/master/HOCON.md

## Additional design notes

I went through the process of writing config handling for several applications, both web and command line, before sitting down and asking myself what my ideal minimal config library API would look like.

An initial iteration worked a bit like [conflate][conflate], being based on merge operations: I'd set up a struct full of defaults, then load a struct from a config file and merge the two, then load environment variables and merge them, and finally tweak based on command line arguments. It ended up being a lot of code, and I quickly realized that I often wanted to merge _some_ values from the environment, but not _all_ of them. Special-casing that quickly made the code a mess. It still exists in a deployed app, but I plan to rip and replace with this library.

Next I tried a method-chaining approach to building an API, `config.For("MyApp").Defaults(some_struct_or_map).With("config.toml")` and so on. That becomes too verbose to be very readable when you want to say "this environment variable corresponds to this flag, this one to this other flag".

Then I tried something like the current approach of resolving a list of possibilities in order, but with type safety everywhere. That became messy because environment variables are never going to be typesafe, so I was faced with the possibility of having methods `GetenvInt`, `GetenvString`, and so on. Then it came to integrating `flag`, and I suddenly realized that it made more sense to put the `flag` package in control, and supply it with a default based on the other sources of configuration.

A common Unix feature is to have a `-config` argument which specifies a particular config file instead of the regular one. I've done that before, but in practice I haven't used the feature much. Rather than `-config deploy.toml` or `-config develop.toml`, it turned out to be more pleasant to make the application detect for itself whether it was in a cloud deployment environment or not. In situations where I needed to override, setting an environment variable wasn't significantly more work than adding a command line flag.

[conflate]: https://github.com/the4thamigo-uk/conflate
