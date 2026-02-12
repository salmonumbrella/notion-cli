package cmd

import "github.com/spf13/pflag"

// flagAlias registers a hidden flag alias that shares the same underlying value.
// This allows shorter flag names (e.g. --props for --properties) without
// duplicating the variable binding. The alias is hidden from help output.
func flagAlias(fs *pflag.FlagSet, name, alias string) {
	f := fs.Lookup(name)
	if f == nil {
		return
	}
	fs.AddFlag(&pflag.Flag{
		Name:        alias,
		Usage:       f.Usage,
		Value:       f.Value,
		DefValue:    f.DefValue,
		NoOptDefVal: f.NoOptDefVal,
		Hidden:      true,
	})
}
