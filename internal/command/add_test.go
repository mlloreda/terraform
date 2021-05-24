package command

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
)

// simple test cases with a simple resource schema
func TestAdd_basic(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("add/basic"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"ami":   {Type: cty.String, Optional: true, Description: "the ami to use"},
						"value": {Type: cty.String, Required: true, Description: "a value of a thing"},
					},
				},
			},
		},
	}

	overrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"):                                providers.FactoryFixed(p),
			addrs.NewProvider("registry.terraform.io", "happycorp", "test"): providers.FactoryFixed(p),
		},
	}

	t.Run("basic", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_instance.new"}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)
		expected := `resource "test_instance" "new" {
  value = <REQUIRED string>
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})

	t.Run("optionals", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-optional", "test_instance.new"}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)
		expected := `resource "test_instance" "new" {
  ami = <OPTIONAL string>
  id = <OPTIONAL string>
  value = <REQUIRED string>
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})

	t.Run("defaults", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-defaults", "test_instance.new"}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)
		expected := `resource "test_instance" "new" {
  value = null
}
`
		if !cmp.Equal(output.Stdout(), expected) {
			fmt.Println(output.Stdout())
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})

	t.Run("alternate provider for resource", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-provider=happycorp/test", "-defaults", "test_instance.new"}
		code := c.Run(args)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)

		// The provider happycorp/test has a localname "othertest" in the provider configuration.
		expected := `resource "test_instance" "new" {
  provider = othertest
  value    = null
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})

	t.Run("resource exists error", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_instance.exists"}
		code := c.Run(args)
		if code != 1 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		output := done(t)
		if !strings.Contains(output.Stderr(), "The resource test_instance.exists is already in this configuration") {
			t.Fatalf("missing expected error message: %s", output.Stderr())
		}
	})

	t.Run("provider not in configuration", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"toast_instance.new"}
		code := c.Run(args)
		if code != 1 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		output := done(t)
		if !strings.Contains(output.Stderr(), "No schema found for provider registry.terraform.io/hashicorp/toast.") {
			t.Fatalf("missing expected error message: %s", output.Stderr())
		}
	})

	t.Run("no schema for resource", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_pet.meow"}
		code := c.Run(args)
		if code != 1 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		output := done(t)
		if !strings.Contains(output.Stderr(), "No resource schema found for test_pet.") {
			t.Fatalf("missing expected error message: %s", output.Stderr())
		}
	})
}

func TestAdd(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("add/module"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// a simple hashicorp/test provider, and a more complex happycorp/test provider
	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Required: true},
					},
				},
			},
		},
	}

	happycorp := testProvider()
	happycorp.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":    {Type: cty.String, Optional: true, Computed: true},
						"ami":   {Type: cty.String, Optional: true, Description: "the ami to use"},
						"value": {Type: cty.String, Required: true, Description: "a value of a thing"},
						"disks": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingList,
								Attributes: map[string]*configschema.Attribute{
									"size":        {Type: cty.String, Optional: true},
									"mount_point": {Type: cty.String, Required: true},
								},
							},
							Optional: true,
						},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"network_interface": {
							Nesting:  configschema.NestingList,
							MinItems: 1,
							Block: configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"device_index": {Type: cty.String, Optional: true},
									"description":  {Type: cty.String, Optional: true},
								},
							},
						},
					},
				},
			},
		},
	}
	providerSource, psClose := newMockProviderSource(t, map[string][]string{
		"registry.terraform.io/happycorp/test": {"1.0.0"},
		"registry.terraform.io/hashicorp/test": {"1.0.0"},
	})
	defer psClose()

	overrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewProvider("registry.terraform.io", "happycorp", "test"): providers.FactoryFixed(happycorp),
			addrs.NewDefaultProvider("test"):                                providers.FactoryFixed(p),
		},
	}

	// the test fixture uses a module, so we need to run init.
	m := Meta{
		testingOverrides: overrides,
		ProviderSource:   providerSource,
		Ui:               new(cli.MockUi),
	}

	init := &InitCommand{
		Meta: m,
	}

	code := init.Run([]string{})
	if code != 0 {
		t.Fatal("init failed")
	}

	t.Run("optional", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-optional", "test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  ami = <OPTIONAL string>
  disks = <OPTIONAL list of object>
  id = <OPTIONAL string>
  value = <REQUIRED string>
  network_interface {
    description = <OPTIONAL string>
    device_index = <OPTIONAL string>
  }
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}

	})

	t.Run("chooses correct provider for root module", func(t *testing.T) {
		// in the root module of this test fixture, "test" is the local name for "happycorp/test"
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  value = <REQUIRED string>
  network_interface {
  }
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})

	t.Run("chooses correct provider for child module", func(t *testing.T) {
		// in the child module of this test fixture, "test" is a default "hashicorp/test" provider
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"module.child.test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  id = <REQUIRED string>
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})

	t.Run("chooses correct provider for an unknown module", func(t *testing.T) {
		// it's weird but ok to use a new/unknown module name; terraform will
		// fall back on default providers (unless a -provider argument is
		// supplied)
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"module.madeup.test_instance.new"}
		code := c.Run(args)
		output := done(t)
		if code != 0 {
			t.Fatalf("wrong exit status. Got %d, want 0", code)
		}

		expected := `resource "test_instance" "new" {
  id = <REQUIRED string>
}
`

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})
}
