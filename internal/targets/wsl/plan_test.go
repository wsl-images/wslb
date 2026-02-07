package wsl

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSwapPlan(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	p := BuildSwapPlan("Demo", true, now)
	if p.StableName != "Demo" {
		t.Fatalf("stable name=%q", p.StableName)
	}
	if p.CandidateName != "Demo__candidate" {
		t.Fatalf("candidate name=%q", p.CandidateName)
	}
	if !strings.HasPrefix(p.BackupName, "Demo__backup__20260102030405") {
		t.Fatalf("backup name=%q", p.BackupName)
	}
	if !p.AllowSwap {
		t.Fatalf("expected allow swap")
	}
	if p.SafetyGate == "" {
		t.Fatalf("expected safety gate text")
	}
}
