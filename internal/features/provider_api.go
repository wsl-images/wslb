package features

import "context"

type DescribeProvider interface {
	Provider
	Describe(context.Context, Ref) (Metadata, error)
}

type ScriptSpec struct {
	Name     string
	Path     string
	Env      map[string]string
	Metadata Metadata
}

type ResolveProvider interface {
	Provider
	Resolve(context.Context, Ref, map[string]interface{}) (ScriptSpec, error)
}
