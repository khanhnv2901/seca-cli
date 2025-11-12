package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type ExternalCheckerConfig struct {
	Name           string
	Command        string
	Args           []string
	Env            map[string]string
	TimeoutSeconds int
}

type ExternalChecker struct {
	name    string
	command string
	args    []string
	env     map[string]string
	timeout time.Duration
}

func NewExternalChecker(cfg ExternalCheckerConfig) *ExternalChecker {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &ExternalChecker{
		name:    cfg.Name,
		command: cfg.Command,
		args:    cfg.Args,
		env:     cfg.Env,
		timeout: timeout,
	}
}

func (e *ExternalChecker) Name() string {
	return e.name
}

func (e *ExternalChecker) Check(ctx context.Context, target string) CheckResult {
	result := CheckResult{
		Target:    target,
		CheckedAt: time.Now().UTC(),
		Status:    "error",
		Error:     "external checker not configured",
	}

	if e.command == "" {
		result.Error = "external checker command is empty"
		return result
	}

	checkCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	args := append([]string{}, e.args...)
	args = append(args, target)

	cmd := exec.CommandContext(checkCtx, e.command, args...)
	cmd.Env = os.Environ()
	if len(e.env) > 0 {
		for k, v := range e.env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			result.Error = string(exitErr.Stderr)
		} else {
			result.Error = err.Error()
		}
		return result
	}

	var pluginResult CheckResult
	if err := json.Unmarshal(output, &pluginResult); err != nil {
		result.Error = fmt.Sprintf("invalid plugin output: %v", err)
		return result
	}

	if pluginResult.Target == "" {
		pluginResult.Target = target
	}
	if pluginResult.CheckedAt.IsZero() {
		pluginResult.CheckedAt = time.Now().UTC()
	}

	return pluginResult
}
