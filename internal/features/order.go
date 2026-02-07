package features

import "fmt"

type Metadata struct {
	ID            string
	Name          string
	Version       string
	Description   string
	Options       map[string]Option
	DependsOn     map[string]map[string]interface{}
	InstallsAfter []string
}

type Option struct {
	Type        string      `json:"type"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Proposals   []string    `json:"proposals,omitempty"`
}

func Order(assignments []string, metas map[string]Metadata) ([]string, error) {
	inDegree := map[string]int{}
	adj := map[string][]string{}
	for _, id := range assignments {
		inDegree[id] = 0
	}

	for _, id := range assignments {
		meta, ok := metas[id]
		if !ok {
			continue
		}
		for dep := range meta.DependsOn {
			if _, ok := inDegree[dep]; ok {
				inDegree[id]++
				adj[dep] = append(adj[dep], id)
			}
		}
		for _, soft := range meta.InstallsAfter {
			if _, ok := inDegree[soft]; ok {
				inDegree[id]++
				adj[soft] = append(adj[soft], id)
			}
		}
	}

	q := make([]string, 0)
	for id, d := range inDegree {
		if d == 0 {
			q = append(q, id)
		}
	}

	out := make([]string, 0, len(assignments))
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		out = append(out, n)
		for _, nei := range adj[n] {
			inDegree[nei]--
			if inDegree[nei] == 0 {
				q = append(q, nei)
			}
		}
	}

	if len(out) != len(assignments) {
		return nil, fmt.Errorf("feature dependency cycle detected")
	}
	return out, nil
}
