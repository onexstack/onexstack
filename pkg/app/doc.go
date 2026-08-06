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
// so that framework setup automatically propagates to subcommands when
// EnableTraverseRunHooks is enabled — ideal for multi-command CLI tools.
//
// Basic usage (server):
//
//	func main() {
//	    opts := options.NewServerOptions()
//	    app.NewApp("myserver", "My Server",
//	        app.WithOptions(opts),
//	        app.WithSlogOptions(opts.SlogOptions),
//	        app.WithVersionInfo(version.Get()),
//	        app.WithRunFunc(func(ctx context.Context) error {
//	            cfg, _ := opts.Config()
//	            server, _ := cfg.New(ctx)
//	            return server.Run(ctx)
//	        }),
//	        app.WithConfigSearchPaths(cli.SearchDirs(".myapp")),
//	    ).Run()
//	}
//
// With lifecycle hooks:
//
//	app.NewApp("myserver", "My Server",
//	    app.WithRunFunc(serverRunFunc),
//	    app.WithPreRunHook(func(ctx context.Context) error {
//	        return initExternalConnections(ctx)
//	    }),
//	    app.WithPreShutdownHook(func(ctx context.Context) error {
//	        return drainConnectionPools(ctx)
//	    }),
//	    app.WithShutdownTimeout(30 * time.Second),
//	).Run()
package app
