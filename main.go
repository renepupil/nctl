package main

import (
	"context"
	"fmt"

	"os"
	"os/signal"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/ninech/nctl/api"
	"github.com/ninech/nctl/apply"
	"github.com/ninech/nctl/auth"
	"github.com/ninech/nctl/create"
	"github.com/ninech/nctl/delete"
	"github.com/ninech/nctl/get"
	"github.com/ninech/nctl/logs"
	"github.com/ninech/nctl/update"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

type flags struct {
	Namespace     string           `help:"Limit commands to a namespace." short:"n"`
	APICluster    string           `help:"Context name of the API cluster." default:"nineapis.ch"`
	LogAPIAddress string           `help:"Address of the deplo.io logging API server." default:"https://logs.deplo.io"`
	Version       kong.VersionFlag `name:"version" help:"Print version information and quit."`
}

type rootCommand struct {
	flags
	Get         get.Cmd                      `cmd:"" help:"Get resource."`
	Auth        auth.Cmd                     `cmd:"" help:"Authenticate with resource."`
	Completions kongplete.InstallCompletions `cmd:"" help:"Print shell completions."`
	Create      create.Cmd                   `cmd:"" help:"Create resource."`
	Apply       apply.Cmd                    `cmd:"" help:"Apply resource."`
	Delete      delete.Cmd                   `cmd:"" help:"Delete resource."`
	Logs        logs.Cmd                     `cmd:"" help:"Get logs of resource."`
	Update      update.Cmd                   `cmd:"" help:"Update resource."`
}

var version = "dev"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(ctx, cancel)

	nctl := &rootCommand{}
	parser := kong.Must(
		nctl,
		kong.Name("nctl"),
		kong.Description("Interact with Nine API resources. See https://docs.nineapis.ch for the full API docs."),
		kong.UsageOnError(),
		kong.Vars{"version": version},
		kong.BindTo(ctx, (*context.Context)(nil)),
	)

	// completion handling
	kongplete.Complete(parser, kongplete.WithPredictor("file", complete.PredictFiles("*")))

	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	// handle the login/oidc cmds separately as we should not try to get the
	// API client if we're not logged in.
	if strings.HasPrefix(kongCtx.Command(), auth.LoginCmdName) {
		kongCtx.FatalIfErrorf(nctl.Auth.Login.Run(ctx, kongCtx.Model.Name))
		return
	}

	if strings.HasPrefix(kongCtx.Command(), auth.OIDCCmdName) {
		kongCtx.FatalIfErrorf(nctl.Auth.OIDC.Run(ctx, os.Stdout))
		return
	}

	client, err := api.New(ctx, nctl.APICluster, nctl.Namespace, api.LogClient(nctl.LogAPIAddress))
	if err != nil {
		fmt.Println(err)
		fmt.Printf("\nUnable to get API client, are you logged in?\n\nUse `%s %s` to login.\n", kongCtx.Model.Name, auth.LoginCmdName)
		os.Exit(1)
	}

	kongCtx.FatalIfErrorf(kongCtx.Run(ctx, client))
}

func setupSignalHandler(ctx context.Context, cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		defer func() {
			signal.Stop(c)
		}()

		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()
}
