package wsl

import (
	"fmt"
	"time"
)

type SwapPlan struct {
	StableName    string `json:"stableName"`
	CandidateName string `json:"candidateName"`
	BackupName    string `json:"backupName"`
	AllowSwap     bool   `json:"allowSwap"`
	SafetyGate    string `json:"safetyGate"`
}

func BuildSwapPlan(stable string, verified bool, now time.Time) SwapPlan {
	backup := fmt.Sprintf("%s__backup__%s", stable, now.UTC().Format("20060102150405"))
	return SwapPlan{
		StableName:    stable,
		CandidateName: stable + "__candidate",
		BackupName:    backup,
		AllowSwap:     verified,
		SafetyGate:    "candidate verification must pass before stable unregister",
	}
}
