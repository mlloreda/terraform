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

func TestAdd(t *testing.T) {
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
	providerSource, psClose := newMockProviderSource(t, map[string][]string{
		"hashicorp/test": {"1.0.0"},
		"happycorp/test": {"1.0.0"},
	})
	defer psClose()

	overrides := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"):                                providers.FactoryFixed(p),
			addrs.NewProvider("registry.terraform.io", "happycorp", "test"): providers.FactoryFixed(p),
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

	t.Run("basic", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"test_instance.new"}
		code = c.Run(args)
		if code != 0 {
			t.Errorf("wrong exit status. Got %d, want 0", code)
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

	t.Run("verbose", func(t *testing.T) {
		view, done := testView(t)
		c := &AddCommand{
			Meta: Meta{
				testingOverrides: overrides,
				View:             view,
			},
		}
		args := []string{"-verbose", "test_instance.new"}
		code = c.Run(args)
		if code != 0 {
			t.Errorf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)
		expected := `resource "test_instance" "new" {
  # the ami to use
  ami = <OPTIONAL string>

  id = <OPTIONAL string>

  # a value of a thing
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
		code = c.Run(args)
		if code != 0 {
			t.Errorf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)
		expected := `resource "test_instance" "new" {
  value = null
}
`
		if !cmp.Equal(output.Stdout(), expected) {
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
		args := []string{"-provider=happycorp/test", "test_instance.new"}
		code = c.Run(args)
		if code != 0 {
			t.Errorf("wrong exit status. Got %d, want 0", code)
		}
		output := done(t)

		// The provider happycorp/test has a localname "othertest" in the provider configuration.
		expected := `resource "test_instance" "new" {
  provider = othertest
  value = <REQUIRED string>
}
`
		fmt.Println(output.Stdout())

		if !cmp.Equal(output.Stdout(), expected) {
			t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
		}
	})

	t.Run("chooses the correct provider for resource", func(t *testing.T) {

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
		code = c.Run(args)
		if code != 1 {
			t.Errorf("wrong exit status. Got %d, want 0", code)
		}

		output := done(t)
		if !strings.Contains(output.Stderr(), "The resource test_instance.exists is already in this configuration") {
			t.Fatalf("missing expected error message: %s", output.Stderr())
		}
	})

	t.Run("provider not in configuration", func(t *testing.T) {

	})
}
