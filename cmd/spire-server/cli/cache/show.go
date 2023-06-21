package cache

// This entire file should be removed in the final offering

import (
	"flag"
	"fmt"

	"github.com/mitchellh/cli"
	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/spiffe/spire/cmd/spire-server/util"
	commoncli "github.com/spiffe/spire/pkg/common/cli"
	"github.com/spiffe/spire/pkg/common/cliprinter"

	"golang.org/x/net/context"
)

const listEntriesRequestPageSize = 500

// NewShowCommand creates a new "show" subcommand for "entry" command.
func NewShowCommand() cli.Command {
	return newShowCommand(commoncli.DefaultEnv)
}

func newShowCommand(env *commoncli.Env) cli.Command {
	return util.AdaptCommand(env, &showCommand{env: env})
}

type showCommand struct {
	printer cliprinter.Printer
	env *commoncli.Env
}

func (c *showCommand) Name() string {
	return "cache show"
}

func (*showCommand) Synopsis() string {
	return "Displays cached registration entries"
}

func (c *showCommand) AppendFlags(f *flag.FlagSet) {
	cliprinter.AppendFlagWithCustomPretty(&c.printer, f, c.env, prettyPrintShow)
}

// Run executes all logic associated with a single invocation of the
// `spire-server entry show` CLI command
func (c *showCommand) Run(ctx context.Context, env *commoncli.Env, serverClient util.ServerClient) error {
	var entries []*types.Entry
	for _, entry := range entries {
		printEntry(entry)
	}
	return nil
}

func prettyPrintShow(env *commoncli.Env, results ...interface{}) error {
	return nil
}

func printEntry(e *types.Entry) {
	fmt.Printf("Entry ID         : %s\n", printableEntryID(e.Id))
	fmt.Printf("SPIFFE ID        : %s\n", protoToIDString(e.SpiffeId))
	fmt.Printf("Parent ID        : %s\n", protoToIDString(e.ParentId))
	fmt.Printf("Revision         : %d\n", e.RevisionNumber)
	fmt.Printf("\n")
}

func printableEntryID(id string) string {
	if id == "" {
		return "(none)"
	}
	return id
}

// protoToIDString converts a SPIFFE ID from the given *types.SPIFFEID to string
func protoToIDString(id *types.SPIFFEID) string {
	if id == nil {
		return ""
	}
	return fmt.Sprintf("spiffe://%s%s", id.TrustDomain, id.Path)
}