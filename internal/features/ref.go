package features

import (
	"fmt"
	"regexp"
	"strings"
)

type Ref struct {
	Raw      string `json:"raw"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
}

var ociLike = regexp.MustCompile(`^[a-zA-Z0-9._-]+\/[a-zA-Z0-9._\/-]+(:[a-zA-Z0-9._-]+)?(@sha256:[a-f0-9]{64})?$`)

func ParseRef(raw string) (Ref, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Ref{}, fmt.Errorf("feature ref is empty")
	}

	if strings.HasPrefix(raw, "wslb:feature/") {
		id := strings.TrimPrefix(raw, "wslb:feature/")
		if id == "" {
			return Ref{}, fmt.Errorf("internal feature id is empty")
		}
		return Ref{Raw: raw, Provider: "internal", Name: id}, nil
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return Ref{Raw: raw, Provider: "devcontainer", Name: raw}, nil
	}

	if ociLike.MatchString(raw) {
		name := raw
		ver := ""
		if i := strings.LastIndex(raw, ":"); i > strings.LastIndex(raw, "/") {
			name = raw[:i]
			ver = raw[i+1:]
		}
		if ver == "" {
			ver = "latest"
		}
		return Ref{Raw: raw, Provider: "devcontainer", Name: name, Version: ver}, nil
	}

	return Ref{}, fmt.Errorf("unsupported feature ref: %s", raw)
}
