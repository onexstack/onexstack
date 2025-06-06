package automaxprocs

import "go.uber.org/automaxprocs/maxprocs"

func init() {
	// disable the default log in automaxprocs.
	maxprocs.Set(maxprocs.Logger(func(s string, i ...any) {}))
}
