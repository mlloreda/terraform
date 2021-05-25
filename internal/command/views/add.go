package views

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Add is the view interface for the "terraform add" command.
type Add interface {
	Resource(addrs.AbsResourceInstance, *configschema.Block, string, *states.ResourceInstanceObject) error
	Diagnostics(tfdiags.Diagnostics)
}

// NewAdd returns an initialized Validate implementation for the given ViewType.
func NewAdd(vt arguments.ViewType, view *View, args *arguments.Add) Add {
	switch vt {
	case arguments.ViewJSON:
		return &addJSON{
			view:     view,
			optional: args.Optional,
			outPath:  args.OutPath,
		}
	case arguments.ViewHuman:
		return &addHuman{
			view:     view,
			optional: args.Optional,
			outPath:  args.OutPath,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type addJSON struct {
	view     *View
	optional bool
	outPath  string
}

func (v *addJSON) Resource(addr addrs.AbsResourceInstance, schema *configschema.Block, provider string, state *states.ResourceInstanceObject) error {
	//render resources as json
	return nil
}

func (v *addJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type addHuman struct {
	view     *View
	optional bool
	outPath  string
}

func (v *addHuman) Resource(addr addrs.AbsResourceInstance, schema *configschema.Block, provider string, state *states.ResourceInstanceObject) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("resource %q %q {\n", addr.Resource.Resource.Type, addr.Resource.Resource.Name))
	if provider != "" {
		buf.WriteString(strings.Repeat(" ", 2))
		buf.WriteString(fmt.Sprintf("provider = %s\n", provider))
	}

	err := v.writeConfigAttributes(&buf, state, schema.Attributes, 2)
	if err != nil {
		return err
	}

	err = v.writeConfigBlocks(&buf, state, schema.BlockTypes, 2)
	if err != nil {
		return err
	}

	buf.WriteString("}")

	// The output better be valid HCL which can be parsed and formatted.
	formatted := hclwrite.Format([]byte(buf.String()))
	_, err = v.view.streams.Println(string(formatted))

	return err
}

func (v *addHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *addHuman) writeConfigAttributes(buf *bytes.Buffer, state *states.ResourceInstanceObject, attrs map[string]*configschema.Attribute, indent int) error {
	if len(attrs) == 0 {
		return nil
	}

	// Get a list of sorted attribute names so the output will be consistent between runs.
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i := range keys {
		name := keys[i]
		attrS := attrs[name]
		if attrS.Required {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))
			var val cty.Value

			if state != nil && state.Value.Type().HasAttribute(name) {
				val = state.Value.GetAttr(name)
			} else {
				val = attrS.EmptyValue()
			}
			tok := hclwrite.TokensForValue(val)
			_, err := tok.WriteTo(buf)
			if err != nil {
				return err
			}
			if attrS.NestedType != nil {
				buf.WriteString(fmt.Sprintf(" # OPTIONAL %s\n", attrS.NestedType.ImpliedType().FriendlyName()))
			} else {
				buf.WriteString(fmt.Sprintf(" # REQUIRED %s\n", attrS.Type.FriendlyName()))
			}
		} else if attrS.Optional && v.optional {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s = ", name))
			tok := hclwrite.TokensForValue(attrS.EmptyValue())
			_, err := tok.WriteTo(buf)
			if err != nil {
				return err
			}
			if attrS.NestedType != nil {
				buf.WriteString(fmt.Sprintf(" # OPTIONAL %s\n", attrS.NestedType.ImpliedType().FriendlyName()))
			} else {
				buf.WriteString(fmt.Sprintf(" # OPTIONAL %s\n", attrS.Type.FriendlyName()))
			}
		}
	}
	return nil
}

func (v *addHuman) writeConfigBlocks(buf *bytes.Buffer, state *states.ResourceInstanceObject, blocks map[string]*configschema.NestedBlock, indent int) error {
	if len(blocks) == 0 {
		return nil
	}

	// Get a list of sorted block names so the output will be consistent between runs.
	names := make([]string, 0, len(blocks))
	for k := range blocks {
		names = append(names, k)
	}
	sort.Strings(names)

	for i := range names {
		name := names[i]
		blockS := blocks[name]

		if blockS.MinItems > 0 {
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString(fmt.Sprintf("%s {\n", name))
			if len(blockS.Attributes) > 0 {
				v.writeConfigAttributes(buf, state, blockS.Attributes, indent+2)
			}
			if len(blockS.BlockTypes) > 0 {
				v.writeConfigBlocks(buf, state, blockS.BlockTypes, indent+2)
			}
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("}\n")
		}
	}
	return nil
}
