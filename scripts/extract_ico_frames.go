package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run scripts/extract_ico_frames.go <input.ico> <output-dir>")
		os.Exit(2)
	}
	input := os.Args[1]
	outputDir := os.Args[2]

	data, err := os.ReadFile(input)
	if err != nil {
		panic(err)
	}
	if len(data) < 6 {
		panic("ico too small")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		panic(err)
	}

	for i := 0; i < count; i++ {
		o := 6 + (16 * i)
		if o+16 > len(data) {
			continue
		}
		size := int(binary.LittleEndian.Uint32(data[o+8 : o+12]))
		off := int(binary.LittleEndian.Uint32(data[o+12 : o+16]))
		if size <= 0 || off < 0 || off+size > len(data) {
			continue
		}
		out := filepath.Join(outputDir, fmt.Sprintf("frame-%02d.png", i))
		if err := os.WriteFile(out, data[off:off+size], 0o644); err != nil {
			panic(err)
		}
		fmt.Println(out)
	}
}
