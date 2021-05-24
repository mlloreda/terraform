package views

import (
	"fmt"
	"strings"

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

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("resource %q %q {\n", addr.Resource.Resource.Type, addr.Resource.Resource.Name))
	writeConfigAttributes(&buf, schema.Attributes, 2)
	writeConfigBlocks(&buf, schema.BlockTypes, 2)
	buf.WriteString("}\n")

	v.view.streams.Println(strings.TrimSpace(buf.String()))

	return nil
}

func (v *addHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

// TODO: besides just general enhancements, honor the flags!
func writeConfigAttributes(buf *strings.Builder, attrs map[string]*configschema.Attribute, indent int) {
	if len(attrs) == 0 {
		return
	}
	for name, attrS := range attrs {
		if attrS.Required || attrS.Optional {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("# %s\n", attrS.Description))
			buf.WriteString(strings.Repeat(" ", indent))
		}
		if attrS.Required {
			buf.WriteString(fmt.Sprintf("%s = <REQUIRED %s>\n\n", name, attrS.Type.FriendlyName()))
		} else if attrS.Optional {
			buf.WriteString(fmt.Sprintf("%s = <OPTIONAL %s>\n\n", name, attrS.Type.FriendlyName()))
		}
	}
}

func writeConfigBlocks(buf *strings.Builder, blocks map[string]*configschema.NestedBlock, indent int) {
	if len(blocks) == 0 {
		return
	}
	for name, blockS := range blocks {
		if blockS.MinItems > 0 {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {", name))
			if len(blockS.Attributes) > 0 {
				writeConfigAttributes(buf, blockS.Attributes, indent+2)
			}
			if len(blockS.BlockTypes) > 0 {
				writeConfigBlocks(buf, blockS.BlockTypes, indent+2)
			}
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("}\n")
		}
	}
}
