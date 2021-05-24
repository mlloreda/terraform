package arguments

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Add represents the command-line arguments for the Add command.
type Add struct {
	// Addr specifies which resource to generate configuration for.
	Addr addrs.AbsResourceInstance

	// ImportID specifies the import ID of an existing resource to import
	// attribute values from.
	ImportID string

	// OutPath contains an optional path to store the generated configuration.
	OutPath string

	// Optional specifies whether or not to include optional attributes in the
	// generated configuration. Defaults to false.
	Optional bool

	// Defaults specifies whether or not to include default "zero" values
	// for the attributes. Defaults to false.
	Defaults bool

	// Provider specifies the provider for the target.
	Provider addrs.Provider

	// ViewType specifies which output format to use
	ViewType ViewType
}

func ParseAdd(args []string) (*Add, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	add := &Add{}

	var jsonOutput bool
	var provider string

	cmdFlags := defaultFlagSet("add")
	cmdFlags.BoolVar(&add.Defaults, "defaults", false, "include default (zero) attribute values")
	cmdFlags.StringVar(&add.ImportID, "from-existing-resource", "", "fill attribute values from an existing resource")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	cmdFlags.BoolVar(&add.Optional, "optional", false, "include optional attributes")
	cmdFlags.StringVar(&add.OutPath, "out", "", "out")
	cmdFlags.StringVar(&provider, "provider", "", "provider")

	if err := cmdFlags.Parse(args); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			err.Error(),
		))
		return add, diags
	}

	switch {
	case jsonOutput:
		add.ViewType = ViewJSON
	default:
		add.ViewType = ViewHuman
	}

	if provider != "" {
		absProvider, providerDiags := addrs.ParseProviderSourceString(provider)
		if providerDiags.HasErrors() {
			// The diagnostics returned from ParseProviderSourceString are
			// specific to the "source" attribute and not suitable for this use
			// case.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid provider string: %s", provider),
				`The "provider" argument must be in the format "[hostname/][namespace/]name"`,
			))
			return add, diags
		}
		add.Provider = absProvider
	}

	args = cmdFlags.Args()

	if len(args) == 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too few command line arguments",
			"Expected exactly one positional argument.",
		))
		return add, diags
	}

	if len(args) > 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Too many command line arguments",
			"Expected exactly one positional argument.",
		))
		return add, diags
	}

	// parse address from the argument
	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(args[0])
	if addrDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Error parsing resource address: %s", args[0]),
			"This command requires that the address references one specific resource instance.",
		))
		return add, diags
	}
	add.Addr = addr

	return add, diags
}
