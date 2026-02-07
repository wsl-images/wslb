package output

import "time"

type Step struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	EndedAt     *time.Time             `json:"endedAt,omitempty"`
	Detail      string                 `json:"detail,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

type Artifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Error struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
}

type Result struct {
	OK         bool       `json:"ok"`
	Command    string     `json:"command"`
	ImageID    string     `json:"imageId,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	EndedAt    time.Time  `json:"endedAt"`
	DurationMS int64      `json:"durationMs"`
	Steps      []Step     `json:"steps,omitempty"`
	Artifacts  []Artifact `json:"artifacts,omitempty"`
	Errors     []Error    `json:"errors,omitempty"`
}

type Event struct {
	TS      time.Time              `json:"ts"`
	Level   string                 `json:"level"`
	Command string                 `json:"command"`
	ImageID string                 `json:"imageId,omitempty"`
	Event   string                 `json:"event"`
	StepID  string                 `json:"stepId,omitempty"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
