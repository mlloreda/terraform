package arguments

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseAdd(t *testing.T) {
	// need a pointer value for the -from-resource-addr tests
	fromResource := mustResourceInstanceAddr("test_foo.bar")

	tests := map[string]struct {
		args      []string
		want      *Add
		wantError string
	}{
		"defaults": {
			[]string{"test_foo.bar"},
			&Add{
				Addr:     mustResourceInstanceAddr("test_foo.bar"),
				ViewType: ViewHuman,
			},
			``,
		},
		"some flags": {
			[]string{"-optional=true", "-json", "test_foo.bar"},
			&Add{
				Addr:     mustResourceInstanceAddr("test_foo.bar"),
				Optional: true,
				ViewType: ViewJSON,
			},
			``,
		},
		"-from-existing-resource": {
			[]string{"-from-existing-resource=test_foo.bar", "module.foo.test_foo.baz"},
			&Add{
				Addr:             mustResourceInstanceAddr("module.foo.test_foo.baz"),
				ViewType:         ViewHuman,
				FromResourceAddr: &fromResource,
			},
			``,
		},
		"-provider": {
			[]string{"-provider=example.com/happycorp/test", "test_foo.bar"},
			&Add{
				Addr:     mustResourceInstanceAddr("test_foo.bar"),
				ViewType: ViewHuman,
				Provider: addrs.NewProvider("example.com", "happycorp", "test"),
			},
			``,
		},

		// Error cases
		"missing required argument": {
			nil,
			&Add{ViewType: ViewHuman},
			`Too few command line arguments`,
		},
		"too many arguments": {
			[]string{"-from-existing-resource=resource_foo.baz", "resource_foo.bar", "module.foo.resource_foo.baz"},
			&Add{
				ViewType: ViewHuman,
			},
			`Too many command line arguments`,
		},
		"invalid target address": {
			[]string{"definitely-not_a-VALID-resource"},
			&Add{ViewType: ViewHuman},
			`Error parsing resource address: definitely-not_a-VALID-resource`,
		},
		"invalid provider flag": {
			[]string{"-provider=/this/isn't/quite/correct", "resource_foo.bar"},
			&Add{ViewType: ViewHuman},
			`Invalid provider string: /this/isn't/quite/correct`,
		},
		"resource type mismatch": {
			[]string{"-from-existing-resource=test_foo.bar", "test_compute.bar"},
			&Add{ViewType: ViewHuman,
				Addr:             mustResourceInstanceAddr("test_compute.bar"),
				FromResourceAddr: &fromResource,
			},
			`Resource type mismatch`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseAdd(test.args)
			if test.wantError != "" {
				if len(diags) != 1 {
					t.Fatalf("got %d diagnostics; want exactly 1\n", len(diags))
				}
				if diags[0].Severity() != tfdiags.Error {
					t.Fatalf("got a warning; want an error\n%s", diags.ErrWithWarnings())
				}
				if desc := diags[0].Description(); desc.Summary != test.wantError {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", desc.Summary, test.wantError)
				}
			} else {
				if len(diags) != 0 {
					t.Fatalf("got %d diagnostics; want none\n%s", len(diags), diags.Err().Error())
				}
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func mustResourceInstanceAddr(s string) addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}
