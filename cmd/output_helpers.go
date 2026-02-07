package cmd

import (
	"fmt"
	"time"

	"github.com/wsl-images/wslb/internal/output"
)

func emitter() *output.Emitter {
	return output.NewEmitter(jsonOut, ndjsonOut)
}

func emitSimpleError(command, imageID string, started time.Time, err error) {
	res := output.NewResult(command, imageID, started, nil, nil, []output.Error{{
		Code:    "WSLB_COMMAND_FAILED",
		Message: err.Error(),
	}})
	emitter().EmitResult(res)
}

func emitValidationIssues(command string, started time.Time, issues []string) {
	errs := make([]output.Error, 0, len(issues))
	for _, i := range issues {
		errs = append(errs, output.Error{Code: "WSLB_VALIDATION", Message: i})
	}
	emitter().EmitResult(output.NewResult(command, "", started, nil, nil, errs))
}

func requireArgAllOrID(all bool, args []string) (string, error) {
	if all {
		return "", nil
	}
	if len(args) < 1 {
		return "", fmt.Errorf("image id is required unless --all is set")
	}
	return args[0], nil
}
