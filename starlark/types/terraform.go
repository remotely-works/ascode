package types

import (
	"fmt"

	"github.com/hashicorp/terraform/version"
	"github.com/mcuadros/ascode/terraform"
	"go.starlark.net/starlark"
)

// Terraform is a representation of Terraform as a starlark.Value
//
//   outline: types
//     types:
//       Terraform
//         Terraform holds all the configuration defined by an script. A global
//         variable called `tf` holds the unique instance of Terraform.
//
//         examples:
//           tf_overview.star
//
//         fields:
//           version string
//             Terraform version.
//           backend Backend
//             Backend used to store the state, if None a `local` backend it's
//             used.
//           provider ProviderCollection
//             Dict with all the providers defined by provider type.
//
//         methods:
//           provider(type, version="", name="") Provider
//             Returns a new provider instance of the given type.
//
//             examples:
//               tf_provider.star
//                 Defining multiple providers and traversing `tf.providers`
//
//             params:
//               type string
//                 Provider type. Eg.: `aws`
//               version string
//                 Version [contraint](https://www.terraform.io/docs/configuration/providers.html#provider-versions),
//                 if `None` latest version available it's used.
//               name string
//                 Local name of the resource, if `None` is provided it's
//                 autogenerated.
//
type Terraform struct {
	b *Backend
	p *ProviderCollection
}

// MakeTerraform returns a new instance of Terraform
func MakeTerraform(pm *terraform.PluginManager) *Terraform {
	return &Terraform{
		p: NewProviderCollection(pm),
	}
}

// Attr honors the starlark.HasAttrs interface.
func (t *Terraform) Attr(name string) (starlark.Value, error) {
	switch name {
	case "version":
		return starlark.String(version.String()), nil
	case "provider":
		return t.p, nil
	case "backend":
		if t.b == nil {
			return starlark.None, nil
		}

		return t.b, nil
	}

	return starlark.None, nil
}

// SetField honors the starlark.HasSetField interface.
func (t *Terraform) SetField(name string, val starlark.Value) error {
	if name != "backend" {
		errmsg := fmt.Sprintf("terraform has no .%s field or method", name)
		return starlark.NoSuchAttrError(errmsg)
	}

	if b, ok := val.(*Backend); ok {
		t.b = b
		return nil
	}

	return fmt.Errorf("unexpected value %s at %s", val.Type(), name)
}

// AttrNames honors the starlark.HasAttrs interface.
func (t *Terraform) AttrNames() []string {
	return []string{"provider", "backend", "version"}
}

// Freeze honors the starlark.Value interface.
func (t *Terraform) Freeze() {} // immutable

// Hash honors the starlark.Value interface.
func (t *Terraform) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: Terraform")
}

// String honors the starlark.Value interface.
func (t *Terraform) String() string {
	return "terraform"
}

// Truth honors the starlark.Value interface.
func (t *Terraform) Truth() starlark.Bool {
	return t.p.Len() != 0
}

// Type honors the starlark.Value interface.
func (t *Terraform) Type() string {
	return "Terraform"
}

var _ starlark.Value = &Terraform{}
