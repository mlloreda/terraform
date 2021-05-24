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
			view:         view,
			defaults:     args.Defaults,
			optional:     args.Optional,
			descriptions: args.Descriptions,
			outPath:      args.OutPath,
		}
	case arguments.ViewHuman:
		return &addHuman{
			view:         view,
			defaults:     args.Defaults,
			optional:     args.Optional,
			descriptions: args.Descriptions,
			outPath:      args.OutPath,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type addJSON struct {
	view                             *View
	optional, descriptions, defaults bool
	outPath                          string
}

func (v *addJSON) Resource(addr addrs.AbsResourceInstance, schema *configschema.Block, provider string, state *states.ResourceInstanceObject) error {
	//render resources as json
	return nil
}

func (v *addJSON) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

type addHuman struct {
	view                             *View
	optional, descriptions, defaults bool
	outPath                          string
}

func (v *addHuman) Resource(addr addrs.AbsResourceInstance, schema *configschema.Block, provider string, state *states.ResourceInstanceObject) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("resource %q %q {\n", addr.Resource.Resource.Type, addr.Resource.Resource.Name))
	if provider != "" {
		buf.WriteString(strings.Repeat(" ", 2))
		buf.WriteString(fmt.Sprintf("provider = %s\n", provider))
	}

	// don't write a newline after the last attribute if there are no blocks to write.
	finalNewline := false
	if len(schema.BlockTypes) > 0 {
		finalNewline = true
	}

	err := v.writeConfigAttributes(&buf, schema.Attributes, 2, finalNewline)
	if err != nil {
		return err
	}

	err = v.writeConfigBlocks(&buf, schema.BlockTypes, 2)
	if err != nil {
		return err
	}

	buf.WriteString("}")

	// if we filled in default zero values, the output should be valid HCL which can be parsed and formatted.
	if v.defaults {
		formatted := hclwrite.Format([]byte(buf.String()))
		_, err = v.view.streams.Println(string(formatted))
	} else {
		// Since we're not running through Format, we need to add the ending newline.
		buf.WriteString("\n")
		_, err = v.view.streams.Println(strings.TrimSpace(buf.String()))
	}

	return err
}

func (v *addHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *addHuman) writeConfigAttributes(buf *bytes.Buffer, attrs map[string]*configschema.Attribute, indent int, finalNewline bool) error {
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
		if attrS.Required || (attrS.Optional && v.optional) {
			if v.descriptions && attrS.Description != "" {
				buf.WriteString(strings.Repeat(" ", indent))
				buf.WriteString(fmt.Sprintf("# %s\n", attrS.Description))
			}
		}
		if attrS.Required {
			buf.WriteString(strings.Repeat(" ", indent))
			if v.defaults {
				buf.WriteString(fmt.Sprintf("%s = ", name))
				tok := hclwrite.TokensForValue(attrS.EmptyValue())
				_, err := tok.WriteTo(buf)
				if err != nil {
					return err
				}
				buf.WriteString("\n")
			} else {
				if attrS.NestedType == nil {
					buf.WriteString(fmt.Sprintf("%s = <REQUIRED %s>\n", name, attrS.Type.FriendlyName()))
				} else {
					buf.WriteString(fmt.Sprintf("%s = <REQUIRED %s>\n", name, attrS.NestedType.ImpliedType().FriendlyName()))
				}
			}
			// write a second newline after the attribute if there are more
			// attributes to write, or if it is the last attribute and finalNewline
			// is true.
			if i < len(keys)-1 {
				buf.WriteString("\n")
			} else if i == len(keys)-1 {
				if finalNewline {
					buf.WriteString("\n")
				}
			}
		} else if attrS.Optional && v.optional {
			buf.WriteString(strings.Repeat(" ", indent))
			if v.defaults {
				buf.WriteString(fmt.Sprintf("%s = ", name))
				tok := hclwrite.TokensForValue(attrS.EmptyValue())
				_, err := tok.WriteTo(buf)
				if err != nil {
					return err
				}
				buf.WriteString("\n")
			} else {
				if attrS.NestedType == nil {
					buf.WriteString(fmt.Sprintf("%s = <OPTIONAL %s>\n", name, attrS.Type.FriendlyName()))
				} else {
					buf.WriteString(fmt.Sprintf("%s = <OPTIONAL %s>\n", name, attrS.NestedType.ImpliedType().FriendlyName()))
				}
			}
			// write a second newline after the attribute if there are more
			// attributes to write, or if it is the last attribute and finalNewline
			// is true.
			if i < len(keys)-1 {
				buf.WriteString("\n")
			} else if i == len(keys)-1 {
				if finalNewline {
					buf.WriteString("\n")
				}
			}
		}
	}
	return nil
}

func (v *addHuman) writeConfigBlocks(buf *bytes.Buffer, blocks map[string]*configschema.NestedBlock, indent int) error {
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
				finalNewline := false
				if len(blockS.BlockTypes) > 0 {
					finalNewline = true
				}
				v.writeConfigAttributes(buf, blockS.Attributes, indent+2, finalNewline)
			}
			if len(blockS.BlockTypes) > 0 {
				v.writeConfigBlocks(buf, blockS.BlockTypes, indent+2)
			}
			buf.WriteString(strings.Repeat(" ", indent))
			buf.WriteString("}\n")
		}
	}
	return nil
}
