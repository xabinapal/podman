package connection

import (
	"fmt"
	"os"

	"github.com/containers/common/pkg/completion"
	"github.com/containers/common/pkg/config"
	"github.com/containers/common/pkg/report"
	"github.com/containers/podman/v3/cmd/podman/common"
	"github.com/containers/podman/v3/cmd/podman/registry"
	"github.com/containers/podman/v3/cmd/podman/system"
	"github.com/containers/podman/v3/cmd/podman/validate"
	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:     "list [options]",
		Aliases: []string{"ls"},
		Args:    validate.NoArgs,
		Short:   "List destination for the Podman service(s)",
		Long:    `List destination information for the Podman service(s) in podman configuration`,
		Example: `podman system connection list
  podman system connection ls
  podman system connection ls --format=json`,
		ValidArgsFunction: completion.AutocompleteNone,
		RunE:              list,
		TraverseChildren:  false,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: listCmd,
		Parent:  system.ConnectionCmd,
	})

	listCmd.Flags().String("format", "", "Custom Go template for printing connections")
	_ = listCmd.RegisterFlagCompletionFunc("format", common.AutocompleteFormat(namedDestination{}))
}

type namedDestination struct {
	Name string
	config.Destination
}

func list(cmd *cobra.Command, _ []string) error {
	cfg, err := config.ReadCustomConfig()
	if err != nil {
		return err
	}

	if len(cfg.Engine.ServiceDestinations) == 0 {
		return nil
	}

	hdrs := []map[string]string{{
		"Identity": "Identity",
		"Name":     "Name",
		"URI":      "URI",
	}}

	rows := make([]namedDestination, 0)
	for k, v := range cfg.Engine.ServiceDestinations {
		if k == cfg.Engine.ActiveService {
			k += "*"
		}

		r := namedDestination{
			Name: k,
			Destination: config.Destination{
				Identity: v.Identity,
				URI:      v.URI,
			},
		}
		rows = append(rows, r)
	}

	format := "{{.Name}}\t{{.Identity}}\t{{.URI}}\n"
	switch {
	case report.IsJSON(cmd.Flag("format").Value.String()):
		buf, err := registry.JSONLibrary().MarshalIndent(rows, "", "    ")
		if err == nil {
			fmt.Println(string(buf))
		}
		return err
	default:
		if cmd.Flag("format").Changed {
			format = cmd.Flag("format").Value.String()
			format = report.NormalizeFormat(format)
		}
	}
	format = report.EnforceRange(format)

	tmpl, err := report.NewTemplate("list").Parse(format)
	if err != nil {
		return err
	}

	w, err := report.NewWriterDefault(os.Stdout)
	if err != nil {
		return err
	}
	defer w.Flush()

	isTable := report.HasTable(cmd.Flag("format").Value.String())
	if !cmd.Flag("format").Changed || isTable {
		_ = tmpl.Execute(w, hdrs)
	}
	return tmpl.Execute(w, rows)
}
