package workspace

import "strings"

const (
	OOBEModeAuto        = "auto"
	OOBEModePredefined  = "predefined"
	OOBEModeInteractive = "interactive"

	OOBEStrategyHybrid   = "hybrid"
	OOBEStrategyWSLBOnly = "wslb-only"
	OOBEStrategyNative   = "native-only"
)

func DefaultOOBEConfig() *WSLOOBEConfig {
	prompt := true
	return &WSLOOBEConfig{
		Mode:              OOBEModeAuto,
		Strategy:          OOBEStrategyHybrid,
		PromptForPassword: &prompt,
	}
}

func NormalizeOOBEConfig(in *WSLOOBEConfig) *WSLOOBEConfig {
	def := DefaultOOBEConfig()
	if in == nil {
		return def
	}
	out := *def
	if strings.TrimSpace(in.Mode) != "" {
		out.Mode = strings.ToLower(strings.TrimSpace(in.Mode))
	}
	if strings.TrimSpace(in.Strategy) != "" {
		out.Strategy = strings.ToLower(strings.TrimSpace(in.Strategy))
	}
	if in.PromptForPassword != nil {
		v := *in.PromptForPassword
		out.PromptForPassword = &v
	}
	return &out
}

func HasConfiguredUser(cfg *WSLImageConfig) bool {
	if cfg == nil {
		return false
	}
	return strings.TrimSpace(cfg.DefaultUser.Name) != ""
}

func EffectiveOOBEMode(cfg *WSLImageConfig) string {
	norm := NormalizeOOBEConfig(cfg.OOBE)
	switch norm.Mode {
	case OOBEModePredefined, OOBEModeInteractive:
		return norm.Mode
	default:
		if HasConfiguredUser(cfg) {
			return OOBEModePredefined
		}
		return OOBEModeInteractive
	}
}
