package views

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Add is the view interface for the "terraform add" command.
type Add interface {
	Resource(addrs.AbsResourceInstance, *configschema.Block, *states.ResourceInstanceObject) tfdiags.Diagnostic
	Diagnostics(tfdiags.Diagnostics)
}

// NewAdd returns an initialized Validate implementation for the given ViewType.
func NewAdd(vt arguments.ViewType, view *View) Add {
	switch vt {
	case arguments.ViewJSON:
		return &addJSON{view: view}
	case arguments.ViewHuman:
		return &addHuman{view: view}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type addJSON struct {
	view *View
}

func (v *addJSON) Resource(addr addrs.AbsResourceInstance, schema *configschema.Block, state *states.ResourceInstanceObject) tfdiags.Diagnostic {
	//render resources as json
	return nil
}

func (v *addJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type addHuman struct {
	view *View
}

func (v *addHuman) Resource(addr addrs.AbsResourceInstance, schema *configschema.Block, state *states.ResourceInstanceObject) tfdiags.Diagnostic {
	// render resources
	return nil
}

func (v *addHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}
