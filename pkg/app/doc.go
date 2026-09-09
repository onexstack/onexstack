// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

// Package app provides a common framework for building applications in the
// OnexStack ecosystem.
//
// The framework standardizes the application lifecycle:
//
//	config loading → flag parsing → options validation → logging init →
//	signal-aware RunFunc → graceful shutdown hooks
//
// It leverages Cobra's native hook chain (PersistentPreRunE / PreRunE / RunE)
// so that framework setup runs in a deterministic order.
//
// An App is intended to be the single application instance in a process:
// creating more than one App (or running the same App more than once) is not
// supported.
//
// Basic usage (server):
//
//	func main() {
//	    opts := options.NewServerOptions()
//	    app.NewApp("myserver", "My Server",
//	        app.WithOptions(opts),
//	        app.WithSlogOptions(opts.SlogOptions),
//	        app.WithVersionInfo(version.Get()),
//	        app.WithRun(func(ctx context.Context) error {
//	            cfg := server.NewConfig(opts)
//	            cfg.Handler = newHTTPHandler() // 注入业务路由
//	            srv, err := cfg.New(ctx)
//	            if err != nil {
//	                return err
//	            }
//	            return srv.Run(ctx)
//	        }),
//	        app.WithConfigSearchPaths(cli.SearchDirs(".myapp")),
//	    ).Run()
//	}
//
// The WithConfig option is a convenience shorthand for setting both the config
// directory and file name in one call:
//
//	app.NewApp("myserver", "My Server",
//	    app.WithConfig("~/.myapp", "config"),
//	)
//
// To search for config in a directory under the user's home directory while
// keeping the default search order (current dir and /etc/<name>), use
// WithDirInHome:
//
//	app.NewApp("myserver", "My Server",
//	    app.WithDirInHome(".onexai"), // searches $HOME/.onexai
//	)
//
// With lifecycle hooks:
//
//	app.NewApp("myserver", "My Server",
//	    app.WithRun(serverRunFunc),
//	    app.WithPreRunHook(func(ctx context.Context) error {
//	        return initExternalConnections(ctx)
//	    }),
//	    app.WithPreShutdownHook(func(ctx context.Context) error {
//	        return drainConnectionPools(ctx)
//	    }),
//	    app.WithShutdownTimeout(30 * time.Second),
//	).Run()
package app
