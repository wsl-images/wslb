package workspace

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DefaultDevcontainerPath = ".devcontainer/devcontainer.json"
)

type DevcontainerSuperset struct {
	Schema        string                 `json:"$schema,omitempty"`
	Name          string                 `json:"name"`
	Image         string                 `json:"image"`
	Features      map[string]interface{} `json:"features"`
	RemoteUser    string                 `json:"remoteUser"`
	ContainerUser string                 `json:"containerUser"`
	WSLB          *DevcontainerWSLB      `json:"wslb"`
}

type DevcontainerWSLB struct {
	ID             string                 `json:"id,omitempty"`
	Version        int                    `json:"version,omitempty"`
	WorkspaceName  string                 `json:"workspaceName,omitempty"`
	OutputDir      string                 `json:"outputDir,omitempty"`
	ImageID        string                 `json:"imageId,omitempty"`
	Target         string                 `json:"target,omitempty"`
	DisplayName    string                 `json:"displayName,omitempty"`
	Description    string                 `json:"description,omitempty"`
	DistroName     string                 `json:"distroName,omitempty"`
	InstallDir     string                 `json:"installDir,omitempty"`
	Managed        *bool                  `json:"managed,omitempty"`
	User           *WSLDefaultUser        `json:"user,omitempty"`
	DefaultUser    *WSLDefaultUser        `json:"defaultUser,omitempty"`
	State          *WSLStateConfig        `json:"state,omitempty"`
	WSLConf        *WSLConfConfig         `json:"wslconf,omitempty"`
	WSLConfig      *WSLGlobalConfig       `json:"wslconfig,omitempty"`
	Distribution   *WSLDistributionConfig `json:"distribution,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	FeatureSources []FeatureSource        `json:"featureSources,omitempty"`
}

func ParseDevcontainerSuperset(data []byte, manifestPath string) (*Manifest, error) {
	var dc DevcontainerSuperset
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, err
	}
	if strings.TrimSpace(dc.Image) == "" {
		return nil, fmt.Errorf("devcontainer.json must include image")
	}

	wslb := dc.WSLB
	imageID := "devcontainer-wsl"
	if wslb != nil && strings.TrimSpace(wslb.ID) != "" {
		imageID = strings.TrimSpace(wslb.ID)
	} else if wslb != nil && strings.TrimSpace(wslb.ImageID) != "" {
		imageID = strings.TrimSpace(wslb.ImageID)
	} else if wslb != nil && strings.TrimSpace(wslb.DistroName) != "" {
		imageID = slug(strings.TrimSpace(wslb.DistroName))
	} else if strings.TrimSpace(dc.Name) != "" {
		imageID = slug(strings.TrimSpace(dc.Name))
	}

	workspaceName := "devcontainer-wsl"
	if strings.TrimSpace(dc.Name) != "" {
		workspaceName = strings.TrimSpace(dc.Name)
	}
	if wslb != nil && strings.TrimSpace(wslb.WorkspaceName) != "" {
		workspaceName = strings.TrimSpace(wslb.WorkspaceName)
	}

	outputDir := "./.wslb-out"
	if wslb != nil && strings.TrimSpace(wslb.OutputDir) != "" {
		outputDir = strings.TrimSpace(wslb.OutputDir)
	}

	target := "wsl"
	if wslb != nil && strings.TrimSpace(wslb.Target) != "" {
		target = strings.TrimSpace(wslb.Target)
	}

	userName := strings.TrimSpace(dc.RemoteUser)
	if userName == "" {
		userName = strings.TrimSpace(dc.ContainerUser)
	}
	if userName == "" {
		userName = "dev"
	}
	defaultUser := WSLDefaultUser{Name: userName, UID: 1000, GID: 1000}
	if wslb != nil {
		if wslb.User != nil {
			defaultUser = *wslb.User
		}
		if wslb.DefaultUser != nil {
			// Legacy field support.
			defaultUser = *wslb.DefaultUser
		}
	}
	if strings.TrimSpace(defaultUser.Name) == "" {
		defaultUser.Name = userName
	}
	if defaultUser.UID == 0 {
		defaultUser.UID = 1000
	}
	if defaultUser.GID == 0 {
		defaultUser.GID = 1000
	}

	distroName := toDistroName(imageID)
	if wslb != nil && strings.TrimSpace(wslb.DistroName) != "" {
		distroName = strings.TrimSpace(wslb.DistroName)
	}

	installDir := filepath.ToSlash(filepath.Join(outputDir, "distros", distroName))
	if wslb != nil && strings.TrimSpace(wslb.InstallDir) != "" {
		installDir = strings.TrimSpace(wslb.InstallDir)
	}

	managed := true
	if wslb != nil && wslb.Managed != nil {
		managed = *wslb.Managed
	}
	state := &WSLStateConfig{
		Mode:       "windows-dir",
		Path:       filepath.ToSlash(filepath.Join(outputDir, "state", imageID+"-home")),
		MountPoint: "/home",
		FSLabel:    "WSLB_STATE",
	}
	if wslb != nil && wslb.State != nil {
		state = mergeStateConfig(state, wslb.State)
	}

	wslconf := DefaultWSLConfConfig()
	if wslb != nil && wslb.WSLConf != nil {
		wslconf = normalizeWSLConf(*wslb.WSLConf, defaultUser.Name)
	} else {
		wslconf = normalizeWSLConf(wslconf, defaultUser.Name)
	}

	displayName := workspaceName
	if wslb != nil && strings.TrimSpace(wslb.DisplayName) != "" {
		displayName = strings.TrimSpace(wslb.DisplayName)
	}
	description := "Generated from devcontainer.json"
	if wslb != nil && strings.TrimSpace(wslb.Description) != "" {
		description = strings.TrimSpace(wslb.Description)
	}

	tags := []string{"latest"}
	if wslb != nil && len(wslb.Tags) > 0 {
		tags = wslb.Tags
	}

	features, err := convertDevcontainerFeatures(dc.Features)
	if err != nil {
		return nil, err
	}

	featureSources := []FeatureSource{{ID: "devcontainers", Type: "devcontainer"}}
	if wslb != nil && len(wslb.FeatureSources) > 0 {
		featureSources = wslb.FeatureSources
	}
	var wslGlobalConfig *WSLGlobalConfig
	if wslb != nil {
		wslGlobalConfig = wslb.WSLConfig
	}

	return &Manifest{
		Version: 1,
		Workspace: WorkspaceConfig{
			Name:           workspaceName,
			OutputDir:      outputDir,
			FeatureSources: featureSources,
		},
		Images: []Image{
			{
				ID:          imageID,
				Target:      target,
				DisplayName: displayName,
				Description: description,
				Base:        dc.Image,
				Tags:        tags,
				WSL: &WSLImageConfig{
					Managed:      managed,
					DistroName:   distroName,
					InstallDir:   installDir,
					DefaultUser:  defaultUser,
					State:        state,
					WSLConf:      wslconf,
					WSLConfig:    wslGlobalConfig,
					Distribution: normalizeDistribution(wslb, defaultUser),
					Features:     features,
				},
			},
		},
	}, nil
}

func mergeStateConfig(def, custom *WSLStateConfig) *WSLStateConfig {
	if def == nil && custom == nil {
		return nil
	}
	if def == nil {
		clone := *custom
		return &clone
	}
	out := *def
	if custom == nil {
		return &out
	}
	if strings.TrimSpace(custom.Mode) != "" {
		out.Mode = custom.Mode
	}
	if strings.TrimSpace(custom.Path) != "" {
		out.Path = custom.Path
	}
	if strings.TrimSpace(custom.MountPoint) != "" {
		out.MountPoint = custom.MountPoint
	}
	if strings.TrimSpace(custom.FSLabel) != "" {
		out.FSLabel = custom.FSLabel
	}
	return &out
}

func normalizeDistribution(wslb *DevcontainerWSLB, defaultUser WSLDefaultUser) *WSLDistributionConfig {
	if wslb == nil || wslb.Distribution == nil {
		return nil
	}
	dist := *wslb.Distribution
	if dist.OOBE == nil {
		dist.OOBE = &WSLDistributionOOBE{}
	}
	if strings.TrimSpace(dist.OOBE.DefaultName) == "" {
		dist.OOBE.DefaultName = defaultUser.Name
	}
	if dist.Shortcut != nil && dist.Shortcut.Enabled == nil {
		enabled := true
		dist.Shortcut.Enabled = &enabled
	}
	return &dist
}

func normalizeWSLConf(cfg WSLConfConfig, defaultUser string) WSLConfConfig {
	out := cfg
	if out.User == nil {
		out.User = &WSLConfUser{}
	}
	if strings.TrimSpace(out.User.Default) == "" {
		out.User.Default = defaultUser
	}
	if out.Boot == nil {
		out.Boot = &WSLConfBoot{}
	}
	if out.LegacySystemd != nil && out.Boot.Systemd == nil {
		v := *out.LegacySystemd
		out.Boot.Systemd = &v
	}
	if out.Boot.Systemd == nil {
		def := true
		out.Boot.Systemd = &def
	}
	if out.Automount == nil {
		out.Automount = &WSLConfAutomount{}
	}
	if out.LegacyAutoBool != nil && out.Automount.Enabled == nil {
		v := *out.LegacyAutoBool
		out.Automount.Enabled = &v
	}
	if out.Automount.Enabled == nil {
		def := true
		out.Automount.Enabled = &def
	}
	if out.Automount.MountFsTab == nil {
		def := true
		out.Automount.MountFsTab = &def
	}
	return out
}

func convertDevcontainerFeatures(input map[string]interface{}) ([]FeatureAssignment, error) {
	if len(input) == 0 {
		return []FeatureAssignment{}, nil
	}
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]FeatureAssignment, 0, len(keys))
	for _, ref := range keys {
		raw := input[ref]
		opts := map[string]interface{}{}
		switch v := raw.(type) {
		case nil:
			// no options
		case bool:
			if !v {
				continue
			}
		case map[string]interface{}:
			opts = v
		default:
			return nil, fmt.Errorf("feature %q options must be object, boolean, or null", ref)
		}
		out = append(out, FeatureAssignment{
			Ref:     ref,
			Options: opts,
		})
	}
	return out, nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(in string) string {
	s := strings.ToLower(strings.TrimSpace(in))
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "devcontainer-wsl"
	}
	return s
}

func toDistroName(in string) string {
	parts := strings.FieldsFunc(in, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return "DevcontainerWSL"
	}
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	name := strings.Join(parts, "")
	if name == "" {
		return "DevcontainerWSL"
	}
	return name
}
