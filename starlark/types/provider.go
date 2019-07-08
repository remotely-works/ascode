package types

import (
	"fmt"
	"strings"

	"github.com/ascode-dev/ascode/terraform"

	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/providers"
	"go.starlark.net/starlark"
)

func BuiltinProvider(pm *terraform.PluginManager) starlark.Value {
	return starlark.NewBuiltin("provider", func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
		var name, version starlark.String
		switch len(args) {
		case 2:
			var ok bool
			version, ok = args.Index(1).(starlark.String)
			if !ok {
				return nil, fmt.Errorf("resource: expected string, go %s", args.Index(1).Type())
			}
			fallthrough
		case 1:
			var ok bool
			name, ok = args.Index(0).(starlark.String)
			if !ok {
				return nil, fmt.Errorf("provider: expected string, got %s", args.Index(0).Type())
			}
		default:
			return nil, fmt.Errorf("resource: unexpected positional arguments count")
		}

		return MakeProvider(pm, name.GoString(), version.GoString())
	})
}

type Provider struct {
	name     string
	provider *plugin.GRPCProvider
	meta     discovery.PluginMeta

	dataSources *MapSchema
	resources   *MapSchema

	*Resource
}

func MakeProvider(pm *terraform.PluginManager, name, version string) (*Provider, error) {
	cli, meta := pm.Get(name, version)
	rpc, err := cli.Client()
	if err != nil {
		return nil, err
	}

	raw, err := rpc.Dispense(plugin.ProviderPluginName)
	if err != nil {
		return nil, err
	}

	provider := raw.(*plugin.GRPCProvider)
	response := provider.GetSchema()

	defer cli.Kill()
	p := &Provider{
		name:     name,
		provider: provider,
		meta:     meta,

		Resource: MakeResource(name, ProviderKind, response.Provider.Block, nil),
	}

	p.dataSources = NewMapSchema(p, name, DataSourceKind, response.DataSources)
	p.resources = NewMapSchema(p, name, ResourceKind, response.ResourceTypes)

	return p, nil
}

func (p *Provider) String() string {
	return fmt.Sprintf("provider(%q)", p.name)
}

// Type honors the starlark.Value interface. It shadows p.Resource.Type.
func (p *Provider) Type() string {
	return "provider"
}

// Attr honors the starlark.HasAttrs interface.
func (p *Provider) Attr(name string) (starlark.Value, error) {
	switch name {
	case "version":
		return starlark.String(p.meta.Version), nil
	case "data":
		return p.dataSources, nil
	case "resource":
		return p.resources, nil
	}

	return p.Resource.Attr(name)
}

// AttrNames honors the starlark.HasAttrs interface.
func (p *Provider) AttrNames() []string {
	return append(p.Resource.AttrNames(), "data", "resource", "version")
}

type MapSchema struct {
	p *Provider

	prefix      string
	kind        Kind
	schemas     map[string]providers.Schema
	collections map[string]*ResourceCollection
}

func NewMapSchema(p *Provider, prefix string, k Kind, schemas map[string]providers.Schema) *MapSchema {
	return &MapSchema{
		p:           p,
		prefix:      prefix,
		kind:        k,
		schemas:     schemas,
		collections: make(map[string]*ResourceCollection),
	}
}

func (m *MapSchema) String() string {
	return fmt.Sprintf("schemas(%q)", m.prefix)
}

func (m *MapSchema) Type() string {
	return "schemas"
}

func (m *MapSchema) Freeze()               {}
func (m *MapSchema) Truth() starlark.Bool  { return true }
func (m *MapSchema) Hash() (uint32, error) { return 1, nil }
func (m *MapSchema) Name() string          { return m.prefix }

func (m *MapSchema) Attr(name string) (starlark.Value, error) {
	name = m.prefix + "_" + name

	if c, ok := m.collections[name]; ok {
		return c, nil
	}

	if schema, ok := m.schemas[name]; ok {
		m.collections[name] = NewResourceCollection(name, m.kind, schema.Block, m.p.Resource)
		return m.collections[name], nil
	}

	return starlark.None, nil
}

func (s *MapSchema) AttrNames() []string {
	names := make([]string, len(s.schemas))

	var i int
	for k := range s.schemas {
		parts := strings.SplitN(k, "_", 2)
		names[i] = parts[1]
		i++
	}

	return names
}