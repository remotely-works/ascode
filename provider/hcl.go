package provider

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclwrite"
	"go.starlark.net/starlark"
)

type HCLCompatible interface {
	ToHCL(b *hclwrite.Body)
}

func BuiltinToHCL(hcl HCLCompatible, f *hclwrite.File) starlark.Value {
	return starlark.NewBuiltin("to_hcl", func(_ *starlark.Thread, _ *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
		hcl.ToHCL(f.Body())
		return starlark.String(string(f.Bytes())), nil
	})
}

func (s *Provider) ToHCL(b *hclwrite.Body) {
	s.dataSources.ToHCL(b)
	s.resources.ToHCL(b)
}

func (t *MapSchema) ToHCL(b *hclwrite.Body) {
	for _, c := range t.collections {
		c.ToHCL(b)
	}
}

func (r *ResourceCollection) ToHCL(b *hclwrite.Body) {
	for i := 0; i < r.Len(); i++ {
		resource := r.Index(i).(*Resource)
		resource.ToHCL(b)
	}
}

func (r *Resource) ToHCL(b *hclwrite.Body) {
	if len(b.Blocks()) != 0 || len(b.Attributes()) != 0 {
		b.AppendNewline()
	}

	var block *hclwrite.Block
	if r.kind != NestedK {
		block = b.AppendNewBlock(string(r.kind), []string{r.typ, r.name})
	} else {
		block = b.AppendNewBlock(r.typ, nil)
	}

	body := block.Body()
	for k := range r.block.Attributes {
		v, ok := r.values[k]
		if !ok {
			continue
		}

		// TODO(mcuadros): I don't know how to do this properly, meanwhile, this works.
		if c, ok := v.v.(*Computed); ok {
			body.SetAttributeTraversal(k, hcl.Traversal{hcl.TraverseRoot{
				Name: c.String(),
			}})

			continue
		}

		body.SetAttributeValue(k, v.Cty())
	}

	for k := range r.block.BlockTypes {
		v, ok := r.values[k]
		if !ok {
			continue
		}

		if collection, ok := v.Value().(HCLCompatible); ok {
			collection.ToHCL(block.Body())
		}
	}
}