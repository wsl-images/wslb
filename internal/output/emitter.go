package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Emitter struct {
	JSON   bool
	NDJSON bool
	Writer io.Writer
	mu     sync.Mutex
}

func NewEmitter(jsonOut, ndjson bool) *Emitter {
	return &Emitter{
		JSON:   jsonOut,
		NDJSON: ndjson,
		Writer: os.Stdout,
	}
}

func (e *Emitter) EmitEvent(ev Event) {
	if !e.NDJSON {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	b, _ := json.Marshal(ev)
	_, _ = fmt.Fprintln(e.Writer, string(b))
}

func (e *Emitter) EmitResult(res Result) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.JSON {
		enc := json.NewEncoder(e.Writer)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}

	if res.OK {
		_, _ = fmt.Fprintf(e.Writer, "%s: ok (%d ms)\n", res.Command, res.DurationMS)
		for _, st := range res.Steps {
			_, _ = fmt.Fprintf(e.Writer, "  - [%s] %s\n", st.Status, st.Name)
		}
		for _, a := range res.Artifacts {
			_, _ = fmt.Fprintf(e.Writer, "  artifact %s: %s\n", a.Name, a.Path)
		}
		return
	}

	_, _ = fmt.Fprintf(e.Writer, "%s: failed (%d ms)\n", res.Command, res.DurationMS)
	for _, er := range res.Errors {
		if er.Remediation == "" {
			_, _ = fmt.Fprintf(e.Writer, "  - %s: %s\n", er.Code, er.Message)
			continue
		}
		_, _ = fmt.Fprintf(e.Writer, "  - %s: %s\n    remediation: %s\n", er.Code, er.Message, er.Remediation)
	}
}

func NewResult(command, imageID string, started time.Time, steps []Step, artifacts []Artifact, errs []Error) Result {
	ended := time.Now().UTC()
	return Result{
		OK:         len(errs) == 0,
		Command:    command,
		ImageID:    imageID,
		StartedAt:  started.UTC(),
		EndedAt:    ended,
		DurationMS: ended.Sub(started).Milliseconds(),
		Steps:      steps,
		Artifacts:  artifacts,
		Errors:     errs,
	}
}
