package workspace

import (
	"encoding/json"
	"fmt"
	"time"
)

type Manifest struct {
	Version   int             `yaml:"version" json:"version"`
	Workspace WorkspaceConfig `yaml:"workspace" json:"workspace"`
	Images    []Image         `yaml:"images" json:"images"`
}

type WorkspaceConfig struct {
	Name           string          `yaml:"name" json:"name"`
	OutputDir      string          `yaml:"outputDir" json:"outputDir"`
	FeatureSources []FeatureSource `yaml:"featureSources" json:"featureSources"`
}

type FeatureSource struct {
	ID   string `yaml:"id" json:"id"`
	Type string `yaml:"type" json:"type"`
}

type Image struct {
	ID          string          `yaml:"id" json:"id"`
	Target      string          `yaml:"target" json:"target"`
	DisplayName string          `yaml:"displayName" json:"displayName"`
	Description string          `yaml:"description" json:"description"`
	Base        string          `yaml:"base" json:"base"`
	Tags        []string        `yaml:"tags" json:"tags"`
	WSL         *WSLImageConfig `yaml:"wsl" json:"wsl,omitempty"`
}

type WSLImageConfig struct {
	Managed      bool                   `yaml:"managed" json:"managed"`
	DistroName   string                 `yaml:"distroName" json:"distroName"`
	InstallDir   string                 `yaml:"installDir" json:"installDir"`
	DefaultUser  WSLDefaultUser         `yaml:"defaultUser" json:"defaultUser"`
	State        *WSLStateConfig        `yaml:"state" json:"state,omitempty"`
	WSLConf      WSLConfConfig          `yaml:"wslconf" json:"wslconf"`
	WSLConfig    *WSLGlobalConfig       `yaml:"wslconfig,omitempty" json:"wslconfig,omitempty"`
	Distribution *WSLDistributionConfig `yaml:"distribution" json:"distribution,omitempty"`
	Features     []FeatureAssignment    `yaml:"features" json:"features"`
}

type WSLDefaultUser struct {
	Name string `yaml:"name" json:"name"`
	UID  int    `yaml:"uid" json:"uid"`
	GID  int    `yaml:"gid" json:"gid"`
}

type WSLStateConfig struct {
	Mode       string `yaml:"mode" json:"mode"`
	Path       string `yaml:"path" json:"path"`
	MountPoint string `yaml:"mountPoint" json:"mountPoint"`
	FSLabel    string `yaml:"fsLabel" json:"fsLabel"`
}

type WSLConfConfig struct {
	// Legacy shorthand fields:
	//   "wslconf": {"systemd": true, "automount": true}
	LegacySystemd  *bool `yaml:"-" json:"-"`
	LegacyAutoBool *bool `yaml:"-" json:"-"`

	User      *WSLConfUser      `yaml:"user,omitempty" json:"user,omitempty"`
	Boot      *WSLConfBoot      `yaml:"boot,omitempty" json:"boot,omitempty"`
	Automount *WSLConfAutomount `yaml:"automount,omitempty" json:"automount,omitempty"`
	Network   *WSLConfNetwork   `yaml:"network,omitempty" json:"network,omitempty"`
	Interop   *WSLConfInterop   `yaml:"interop,omitempty" json:"interop,omitempty"`
	GPU       *WSLConfGPU       `yaml:"gpu,omitempty" json:"gpu,omitempty"`
	Time      *WSLConfTime      `yaml:"time,omitempty" json:"time,omitempty"`
}

type WSLConfUser struct {
	Default string `yaml:"default,omitempty" json:"default,omitempty"`
}

type WSLConfBoot struct {
	Systemd       *bool  `yaml:"systemd,omitempty" json:"systemd,omitempty"`
	Command       string `yaml:"command,omitempty" json:"command,omitempty"`
	ProtectBinfmt *bool  `yaml:"protectBinfmt,omitempty" json:"protectBinfmt,omitempty"`
}

type WSLConfAutomount struct {
	Enabled     *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Root        string `yaml:"root,omitempty" json:"root,omitempty"`
	Options     string `yaml:"options,omitempty" json:"options,omitempty"`
	MountFsTab  *bool  `yaml:"mountFsTab,omitempty" json:"mountFsTab,omitempty"`
	CrossDistro *bool  `yaml:"crossDistro,omitempty" json:"crossDistro,omitempty"`
}

type WSLConfNetwork struct {
	Hostname           string `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	GenerateHosts      *bool  `yaml:"generateHosts,omitempty" json:"generateHosts,omitempty"`
	GenerateResolvConf *bool  `yaml:"generateResolvConf,omitempty" json:"generateResolvConf,omitempty"`
}

type WSLConfInterop struct {
	Enabled           *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	AppendWindowsPath *bool `yaml:"appendWindowsPath,omitempty" json:"appendWindowsPath,omitempty"`
}

type WSLConfGPU struct {
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

type WSLConfTime struct {
	UseWindowsTimezone *bool `yaml:"useWindowsTimezone,omitempty" json:"useWindowsTimezone,omitempty"`
}

type WSLDistributionConfig struct {
	OOBE            *WSLDistributionOOBE            `yaml:"oobe,omitempty" json:"oobe,omitempty"`
	Shortcut        *WSLDistributionShortcut        `yaml:"shortcut,omitempty" json:"shortcut,omitempty"`
	WindowsTerminal *WSLDistributionWindowsTerminal `yaml:"windowsterminal,omitempty" json:"windowsterminal,omitempty"`
}

type WSLDistributionOOBE struct {
	DefaultName string `yaml:"defaultName,omitempty" json:"defaultName,omitempty"`
	DefaultUID  *int   `yaml:"defaultUid,omitempty" json:"defaultUid,omitempty"`
	Command     string `yaml:"command,omitempty" json:"command,omitempty"`
}

type WSLDistributionShortcut struct {
	Enabled *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Icon    WSLDIcon `yaml:"icon,omitempty" json:"icon,omitempty"`
}

type WSLDistributionWindowsTerminal struct {
	Enabled         *bool           `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ProfileTemplate string          `yaml:"profileTemplate,omitempty" json:"profileTemplate,omitempty"`
	Template        json.RawMessage `yaml:"template,omitempty" json:"template,omitempty"`
}

type WSLDIcon struct {
	Path       string `yaml:"path,omitempty" json:"path,omitempty"`
	SimpleIcon string `yaml:"simpleIcon,omitempty" json:"simpleIcon,omitempty"`
	Color      string `yaml:"color,omitempty" json:"color,omitempty"`
	Style      string `yaml:"style,omitempty" json:"style,omitempty"`
}

type WSLGlobalConfig struct {
	Apply        *bool                  `yaml:"apply,omitempty" json:"apply,omitempty"`
	WSL2         map[string]interface{} `yaml:"wsl2,omitempty" json:"wsl2,omitempty"`
	Experimental map[string]interface{} `yaml:"experimental,omitempty" json:"experimental,omitempty"`
}

type FeatureAssignment struct {
	Ref     string                 `yaml:"ref" json:"ref"`
	Options map[string]interface{} `yaml:"options" json:"options"`
}

func DefaultWSLConfConfig() WSLConfConfig {
	sys := true
	auto := true
	mfstab := true
	return WSLConfConfig{
		Boot: &WSLConfBoot{
			Systemd: &sys,
		},
		Automount: &WSLConfAutomount{
			Enabled:    &auto,
			MountFsTab: &mfstab,
		},
	}
}

func (c *WSLConfConfig) UnmarshalJSON(data []byte) error {
	type alias WSLConfConfig
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var out alias
	if v, ok := raw["user"]; ok {
		var user WSLConfUser
		if err := json.Unmarshal(v, &user); err != nil {
			return fmt.Errorf("wslconf.user: %w", err)
		}
		out.User = &user
	}
	if v, ok := raw["boot"]; ok {
		var boot WSLConfBoot
		if err := json.Unmarshal(v, &boot); err != nil {
			return fmt.Errorf("wslconf.boot: %w", err)
		}
		out.Boot = &boot
	}
	if v, ok := raw["automount"]; ok {
		var legacy bool
		if err := json.Unmarshal(v, &legacy); err == nil {
			out.LegacyAutoBool = &legacy
		} else {
			var auto WSLConfAutomount
			if err := json.Unmarshal(v, &auto); err != nil {
				return fmt.Errorf("wslconf.automount: %w", err)
			}
			out.Automount = &auto
		}
	}
	if v, ok := raw["network"]; ok {
		var network WSLConfNetwork
		if err := json.Unmarshal(v, &network); err != nil {
			return fmt.Errorf("wslconf.network: %w", err)
		}
		out.Network = &network
	}
	if v, ok := raw["interop"]; ok {
		var interop WSLConfInterop
		if err := json.Unmarshal(v, &interop); err != nil {
			return fmt.Errorf("wslconf.interop: %w", err)
		}
		out.Interop = &interop
	}
	if v, ok := raw["gpu"]; ok {
		var gpu WSLConfGPU
		if err := json.Unmarshal(v, &gpu); err != nil {
			return fmt.Errorf("wslconf.gpu: %w", err)
		}
		out.GPU = &gpu
	}
	if v, ok := raw["time"]; ok {
		var timeCfg WSLConfTime
		if err := json.Unmarshal(v, &timeCfg); err != nil {
			return fmt.Errorf("wslconf.time: %w", err)
		}
		out.Time = &timeCfg
	}
	if v, ok := raw["systemd"]; ok {
		var sys bool
		if err := json.Unmarshal(v, &sys); err != nil {
			return fmt.Errorf("wslconf.systemd: %w", err)
		}
		out.LegacySystemd = &sys
	}

	*c = WSLConfConfig(out)
	return nil
}

func (i *WSLDIcon) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		i.Path = asString
		return nil
	}
	type alias WSLDIcon
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*i = WSLDIcon(out)
	return nil
}

func (d *WSLDistributionConfig) UnmarshalJSON(data []byte) error {
	type alias WSLDistributionConfig
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		var out alias
		if err2 := json.Unmarshal(data, &out); err2 != nil {
			return err
		}
		*d = WSLDistributionConfig(out)
		return nil
	}

	var out alias
	if v, ok := raw["oobe"]; ok {
		var oobe WSLDistributionOOBE
		if err := json.Unmarshal(v, &oobe); err != nil {
			return fmt.Errorf("distribution.oobe: %w", err)
		}
		out.OOBE = &oobe
	}
	if v, ok := raw["shortcut"]; ok {
		var shortcut WSLDistributionShortcut
		if err := json.Unmarshal(v, &shortcut); err != nil {
			return fmt.Errorf("distribution.shortcut: %w", err)
		}
		out.Shortcut = &shortcut
	}

	var (
		legacyWT json.RawMessage
		camelWT  json.RawMessage
	)
	if v, ok := raw["windowsterminal"]; ok {
		legacyWT = v
	}
	if v, ok := raw["windowsTerminal"]; ok {
		camelWT = v
	}
	if len(legacyWT) > 0 && len(camelWT) > 0 {
		return fmt.Errorf("distribution: use only one of `windowsterminal` or `windowsTerminal`")
	}
	switch {
	case len(legacyWT) > 0:
		var wt WSLDistributionWindowsTerminal
		if err := json.Unmarshal(legacyWT, &wt); err != nil {
			return fmt.Errorf("distribution.windowsterminal: %w", err)
		}
		out.WindowsTerminal = &wt
	case len(camelWT) > 0:
		var wt WSLDistributionWindowsTerminal
		if err := json.Unmarshal(camelWT, &wt); err != nil {
			return fmt.Errorf("distribution.windowsTerminal: %w", err)
		}
		out.WindowsTerminal = &wt
	}

	*d = WSLDistributionConfig(out)
	return nil
}

func (wt *WSLDistributionWindowsTerminal) UnmarshalJSON(data []byte) error {
	type alias WSLDistributionWindowsTerminal
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		var out alias
		if err2 := json.Unmarshal(data, &out); err2 != nil {
			return err
		}
		*wt = WSLDistributionWindowsTerminal(out)
		return nil
	}
	var out alias
	if v, ok := raw["enabled"]; ok {
		var b bool
		if err := json.Unmarshal(v, &b); err != nil {
			return fmt.Errorf("windowsterminal.enabled: %w", err)
		}
		out.Enabled = &b
	}
	if v, ok := raw["profileTemplate"]; ok {
		if err := json.Unmarshal(v, &out.ProfileTemplate); err != nil {
			return fmt.Errorf("windowsterminal.profileTemplate: %w", err)
		}
	}
	if v, ok := raw["ProfileTemplate"]; ok {
		if err := json.Unmarshal(v, &out.ProfileTemplate); err != nil {
			return fmt.Errorf("windowsterminal.ProfileTemplate: %w", err)
		}
	}
	if v, ok := raw["template"]; ok {
		out.Template = append(out.Template[:0], v...)
	}
	if v, ok := raw["templateJson"]; ok {
		out.Template = append(out.Template[:0], v...)
	}
	if v, ok := raw["Template"]; ok {
		out.Template = append(out.Template[:0], v...)
	}
	*wt = WSLDistributionWindowsTerminal(out)
	return nil
}

type Plan struct {
	WorkspaceName string      `json:"workspaceName"`
	GeneratedAt   time.Time   `json:"generatedAt"`
	Images        []ImagePlan `json:"images"`
}

type ImagePlan struct {
	ImageID          string   `json:"imageId"`
	Target           string   `json:"target"`
	DisplayName      string   `json:"displayName"`
	Prerequisites    []string `json:"prerequisites"`
	Steps            []string `json:"steps"`
	DeterministicDir string   `json:"deterministicDir"`
	CandidateName    string   `json:"candidateName,omitempty"`
	BackupPattern    string   `json:"backupPattern,omitempty"`
	ArtifactPath     string   `json:"artifactPath,omitempty"`
}
