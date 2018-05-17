/*
envflag is a simple wrapper for Go's standard flag.Parse() that overrides sets flags
to the same name (but upper case) environment variable.
*/
package envflag

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// ParseFlagSet walks the specified flag.FlagSet's flags and returns the first error occured (if any)
// If no
func ParseFlagSet(fs *flag.FlagSet, arguments []string) error {
	var exitErr error
	fs.VisitAll(func(f *flag.Flag) {
		err := visitor(f)
		if err != nil && exitErr == nil {
			exitErr = err
		}
	})
	if exitErr != nil {
		return exitErr
	}
	return fs.Parse(arguments)
}

// Parse walks all flags globally registered with the flag package and exits with the first error found (if any)
func Parse() {
	var exitErr error
	flag.VisitAll(func(f *flag.Flag) {
		err := visitor(f)
		if err != nil && exitErr == nil {
			exitErr = err
		}
	})
	flag.Parse() // will exit if there is a bad flag

	// if we got an invalid env var, we never set it, so flag.Parse() will pass.
	if exitErr != nil {
		fmt.Fprintln(os.Stderr, exitErr)
		flag.Usage()
		os.Exit(1)
	}
}

func visitor(f *flag.Flag) error {
	envVar := flagNameToEnvName(f.Name)
	envVal := os.Getenv(envVar)
	if envVal != "" {
		err := f.Value.Set(envVal)
		if err != nil {
			return fmt.Errorf("Invalid environment variable: %s=%s, reason: %s", envVar, envVal, err)
		}
	}
	return nil
}

func flagNameToEnvName(fn string) string {
	fn = strings.ToUpper(fn)
	fn = strings.Replace(fn, "-", "_", -1)
	fn = strings.Replace(fn, ".", "_", -1)
	return fn
}
