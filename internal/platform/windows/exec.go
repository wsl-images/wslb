package windows

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os/exec"
	"strings"
	"unicode/utf16"
)

type CmdResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type Runner struct{}

func NewRunner() *Runner { return &Runner{} }

func (r *Runner) Run(ctx context.Context, bin string, args ...string) (CmdResult, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	res := CmdResult{Command: strings.TrimSpace(bin + " " + strings.Join(args, " "))}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		} else {
			res.ExitCode = 1
		}
		decoded := DecodePossiblyUTF16(out)
		res.Stderr = decoded
		return res, fmt.Errorf("command failed: %w", err)
	}
	res.Stdout = DecodePossiblyUTF16(out)
	return res, nil
}

func DecodePossiblyUTF16(b []byte) string {
	if len(b) >= 2 && ((b[0] == 0xff && b[1] == 0xfe) || (b[0] == 0xfe && b[1] == 0xff)) {
		return decodeUTF16(b)
	}
	if bytes.Contains(b, []byte{0x00}) {
		return decodeUTF16(b)
	}
	return strings.TrimSpace(string(b))
}

func decodeUTF16(b []byte) string {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	var order binary.ByteOrder = binary.LittleEndian
	if len(b) >= 2 && b[0] == 0xfe && b[1] == 0xff {
		order = binary.BigEndian
		b = b[2:]
	} else if len(b) >= 2 && b[0] == 0xff && b[1] == 0xfe {
		b = b[2:]
	}

	u16 := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u16 = append(u16, order.Uint16(b[i:i+2]))
	}
	s := string(utf16.Decode(u16))
	s = strings.ReplaceAll(s, "\x00", "")
	return strings.TrimSpace(s)
}
