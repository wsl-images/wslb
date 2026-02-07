package features

import "fmt"

type Provider interface {
	Kind() string
}

type Router struct {
	providers map[string]Provider
}

func NewRouter(providers ...Provider) *Router {
	m := make(map[string]Provider, len(providers))
	for _, p := range providers {
		m[p.Kind()] = p
	}
	return &Router{providers: m}
}

func (r *Router) ProviderFor(raw string) (Provider, Ref, error) {
	parsed, err := ParseRef(raw)
	if err != nil {
		return nil, Ref{}, err
	}
	p, ok := r.providers[parsed.Provider]
	if !ok {
		return nil, Ref{}, fmt.Errorf("no provider registered for %s", parsed.Provider)
	}
	return p, parsed, nil
}
