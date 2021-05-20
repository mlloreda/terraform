package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// AddCommand is a Command implementation that generates resource configuration templates.
type AddCommand struct {
	Meta
}

func (c *AddCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseAdd(rawArgs)
	view := views.NewAdd(args.ViewType, c.View)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupprted backend",
			ErrUnsupportedLocalOp,
		))
		view.Diagnostics(diags)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error determining current working directory",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.AllowUnsetVariables = true
	opReq.ConfigDir = cwd

	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error initializing config loader",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Get the context
	ctx, _, ctxDiags := local.Context(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// load the configuration and verify that the resource address doesn't
	// already exist in the config.
	// TODO: check + emit an error if the module doesn't exist at all.
	var module *configs.Module
	if args.Addr.Module.IsRoot() {
		module = ctx.Config().Module
	} else {
		module = ctx.Config().Root.Descendent(args.Addr.Module.Module()).Module
	}

	if module == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Module not found",
			fmt.Sprintf("The module %s was not found in the current configuration.", args.Addr.Module.String()),
		))
		c.View.Diagnostics(diags)
		return 1
	}

	if rs, ok := module.ManagedResources[args.Addr.ContainingResource().Config().String()]; ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource already in configuration",
			Detail:   fmt.Sprintf("The resource %s is already in this configuration. Add cannot overwrite existing resources.", args.Addr),
			Subject:  &rs.DeclRange,
		})

		c.View.Diagnostics(diags)
		return 1
	}

	// Get the schemas from the context
	schemas := ctx.Schemas()

	// TODO: This needs to be improved; check for a provider argument + check
	// the configuration for a local before falling back to the implied
	rs := args.Addr.Resource.Resource

	provider := rs.ImpliedProvider()
	absProvider := addrs.ImpliedProviderForUnqualifiedType(provider)

	if _, exists := schemas.Providers[absProvider]; !exists {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Missing schema for provider",
			fmt.Sprintf("No schema found for provider %s. Please verify that this provider exists in the configuration.", absProvider.String()),
		))
	}

	schema, _ := schemas.ResourceTypeConfig(absProvider, rs.Mode, rs.Type)

	diags = diags.Append(view.Resource(args.Addr, schema, nil))

	if diags.HasErrors() {
		return 1
	}

	return 0
}
