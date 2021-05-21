package command

import (
	"fmt"
	"os"
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
						"ami":   {Type: cty.String, Optional: true},
						"value": {Type: cty.String, Required: true},
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
	fmt.Println(output.Stdout())
	expected := ``

	if !cmp.Equal(output.Stdout(), expected) {
		t.Fatalf("wrong output:\n%s", cmp.Diff(output.Stdout(), expected))
	}

	// test cases:
	// basic success
	// in-module success
	// success with provider weirdness (using -provider to override default)
	// success with provider weirdness (gets non-default provider fqn from required_provideres)
	// error: no config / no init
	// error: resource already in config
	// error: referencing module not in config
	// error: provider not in config
}
