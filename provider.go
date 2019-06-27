package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/configs/configschema"

	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/providers"
	"go.starlark.net/starlark"
)

type ProviderInstance struct {
	name     string
	provider *plugin.GRPCProvider

	dataSources *MapSchemaIntance
	resources   *MapSchemaIntance
}

func NewProviderInstance(pm *PluginManager, name string) (*ProviderInstance, error) {
	cli := pm.Get(name, "")

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
	return &ProviderInstance{
		name:        name,
		provider:    provider,
		dataSources: NewMapSchemaInstance(name, response.DataSources),
		resources:   NewMapSchemaInstance(name, response.ResourceTypes),
	}, nil
}

func computeNestedBlocks(s map[string]providers.Schema) map[string]*configschema.NestedBlock {
	blks := make(map[string]*configschema.NestedBlock)
	for k, block := range s {
		for n, nested := range block.Block.BlockTypes {
			key := k + "_" + n
			doComputeNestedBlocks(key, nested, blks)
		}
	}

	return blks
}

func doComputeNestedBlocks(name string, b *configschema.NestedBlock, list map[string]*configschema.NestedBlock) {
	list[name] = b
	for k, block := range b.BlockTypes {
		key := name + "_" + k
		list[key] = block

		doComputeNestedBlocks(key, block, list)
	}
}

func (t *ProviderInstance) String() string {
	return fmt.Sprintf("provider(%q)", t.name)
}

func (t *ProviderInstance) Type() string {
	return "provider-instance"
}

func (t *ProviderInstance) Freeze()               {}
func (t *ProviderInstance) Truth() starlark.Bool  { return true }
func (t *ProviderInstance) Hash() (uint32, error) { return 1, nil }
func (t *ProviderInstance) Name() string          { return t.name }
func (s *ProviderInstance) Attr(name string) (starlark.Value, error) {
	switch name {
	case "data":
		return s.dataSources, nil
	case "resource":
		return s.resources, nil
	}

	return starlark.None, nil
}

func (s *ProviderInstance) AttrNames() []string {
	return []string{"data", "resource"}
}

type MapSchemaIntance struct {
	prefix  string
	schemas map[string]providers.Schema
}

func NewMapSchemaInstance(prefix string, schemas map[string]providers.Schema) *MapSchemaIntance {
	return &MapSchemaIntance{prefix: prefix, schemas: schemas}
}

func (t *MapSchemaIntance) String() string {
	return fmt.Sprintf("schemas(%q)", t.prefix)
}

func (t *MapSchemaIntance) Type() string {
	return "schemas"
}

func (t *MapSchemaIntance) Freeze()               {}
func (t *MapSchemaIntance) Truth() starlark.Bool  { return true }
func (t *MapSchemaIntance) Hash() (uint32, error) { return 1, nil }
func (t *MapSchemaIntance) Name() string          { return t.prefix }

func (s *MapSchemaIntance) Attr(name string) (starlark.Value, error) {
	name = s.prefix + "_" + name

	if schema, ok := s.schemas[name]; ok {
		return NewResourceInstanceConstructor(name, schema.Block, nil), nil
	}

	return starlark.None, nil
}

func (s *MapSchemaIntance) AttrNames() []string {
	names := make([]string, len(s.schemas))

	var i int
	for k := range s.schemas {
		parts := strings.SplitN(k, "_", 2)
		names[i] = parts[1]
		i++
	}

	return names
}
