package get

import (
	"context"
	"fmt"
	"io"

	"github.com/gobuffalo/flect"
	"github.com/ninech/nctl/api"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Cmd struct {
	Output             output                `help:"Configures list output. ${enum}" short:"o" enum:"full,no-header,contexts" default:"full"`
	AllNamespaces      bool                  `help:"apply the get over all namespaces." short:"A"`
	Clusters           clustersCmd           `cmd:"" group:"infrastructure.nine.ch" help:"Get Kubernetes Clusters."`
	APIServiceAccounts apiServiceAccountsCmd `cmd:"" group:"iam.nine.ch" name:"apiserviceaccounts" aliases:"asa" help:"Get API Service Accounts."`
	Applications       applicationsCmd       `cmd:"" group:"deplo.io" name:"applications" aliases:"app,apps" help:"Get deplo.io Applications. (Beta - requires access)"`

	opts []runtimeclient.ListOption
}

type output string

const (
	full     output = "full"
	noHeader output = "no-header"
	contexts output = "contexts"
)

type listOpt func(cmd *Cmd)

func matchName(name string) listOpt {
	return func(cmd *Cmd) {
		if len(name) == 0 {
			return
		}
		cmd.opts = append(cmd.opts, runtimeclient.MatchingFields{"metadata.name": name})
	}
}

func (cmd *Cmd) list(ctx context.Context, client *api.Client, list runtimeclient.ObjectList, opts ...listOpt) error {
	for _, opt := range opts {
		opt(cmd)
	}

	if !cmd.AllNamespaces {
		cmd.opts = append(cmd.opts, runtimeclient.InNamespace(client.Namespace))
	}

	return client.List(ctx, list, cmd.opts...)
}

// writeHeader writes the header row, prepending the namespace row if
// cmd.AllNamespaces is set.
func (cmd *Cmd) writeHeader(w io.Writer, headings ...string) {
	if cmd.AllNamespaces {
		headings = append([]string{"NAMESPACE"}, headings...)
	}
	cmd.writeTabRow(w, "", headings...)
}

// writeTabRow writes a row to w, prepending the namespace if
// cmd.AllNamespaces is set and the namespace is not empty.
func (cmd *Cmd) writeTabRow(w io.Writer, namespace string, row ...string) {
	if cmd.AllNamespaces && len(namespace) != 0 {
		fmt.Fprintf(w, "%s\t", namespace)
	}

	for _, r := range row {
		fmt.Fprintf(w, "%s\t", r)
	}
	fmt.Fprintf(w, "\n")
}

func printEmptyMessage(kind, namespace string) {
	if namespace == "" {
		fmt.Printf("no %s found\n", flect.Pluralize(kind))
		return
	}

	fmt.Printf("no %s found in namespace %s\n", flect.Pluralize(kind), namespace)
}
