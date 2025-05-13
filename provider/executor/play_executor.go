package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sapslaj/mid/provider/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func RunPlaybook(ctx context.Context, connection *types.Connection, playbook []byte) ([]byte, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.RunPlaybook", trace.WithAttributes(
		attribute.String("exec.strategy", "ansible"),
		attribute.String("connection.host", *connection.Host),
	))
	defer span.End()

	dir, err := os.MkdirTemp("", "pulumi-mid")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer os.RemoveAll(dir)

	playbookPath := filepath.Join(dir, "play.yaml")
	err = os.WriteFile(playbookPath, playbook, 0600)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	inventoryVars := map[string]any{
		"ansible_host":            *connection.Host,
		"ansible_ssh_common_args": "-q -o StrictHostKeyChecking=no",
	}
	if connection.Port != nil {
		inventoryVars["ansible_port"] = int(*connection.Port)
	}
	if connection.User != nil {
		inventoryVars["ansible_user"] = *connection.User
	}
	if connection.Password != nil {
		inventoryVars["ansible_password"] = *connection.Password
	}
	if connection.PrivateKey != nil {
		privateKeyPath := filepath.Join(dir, "private-key.pem")
		err = os.WriteFile(privateKeyPath, []byte(*connection.PrivateKey), 0400)
		if err != nil {
			return nil, err
		}
		inventoryVars["ansible_ssh_private_key_file"] = privateKeyPath
	}

	// TODO: proxy support
	// TODO: agentSocketPath support
	// TODO: dialErrorLimit support?
	// TODO: perDialTimeout support?
	// TODO: privateKeyPassword support?

	inventoryData, err := json.Marshal(map[string]any{
		"all": map[string]any{
			"hosts": map[string]any{
				*connection.Host: nil,
			},
			"vars": inventoryVars,
		},
	})

	inventoryPath := filepath.Join(dir, "inventory.yaml")
	err = os.WriteFile(inventoryPath, inventoryData, 0600)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "ansible-playbook", "-i", inventoryPath, playbookPath, "--diff")
	cmd.Env = append(os.Environ(),
		"ANSIBLE_CALLBACK_WHITELIST=json",
		"ANSIBLE_STDOUT_CALLBACK=json",
		"ANSIBLE_SSH_RETRIES=10",
	)
	cmd.Dir = dir

	stderrBuffer := &bytes.Buffer{}
	cmd.Stderr = stderrBuffer
	stdoutBuffer := &bytes.Buffer{}
	cmd.Stdout = stdoutBuffer

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf(
			"command exited with non success code: %d stderr=%s stdout=%s err=%w",
			cmd.ProcessState.ExitCode(),
			stderrBuffer.String(),
			stdoutBuffer.String(),
			err,
		)
		span.SetStatus(codes.Error, err.Error())
		return stdoutBuffer.Bytes(), err
	}

	span.SetStatus(codes.Ok, "")
	return stdoutBuffer.Bytes(), nil
}

type PlayOutputItemMetadataDuration struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type PlayOutputItemMetadata struct {
	Duration PlayOutputItemMetadataDuration `json:"duration"`
	Id       string                         `json:"id"`
	Name     string                         `json:"name"`
	Path     string                         `json:"path"`
}

type PlayOutputTask struct {
	Hosts map[string]any         `json:"hosts"`
	Task  PlayOutputItemMetadata `json:"task"`
}

type PlayOutputResult struct {
	Play  PlayOutputItemMetadata `json:"play"`
	Tasks []PlayOutputTask       `json:"tasks"`
}

type PlayOutputStats struct {
	Changed     uint `json:"changed"`
	Failures    uint `json:"failures"`
	Ignored     uint `json:"ignored"`
	Ok          uint `json:"ok"`
	Rescued     uint `json:"rescued"`
	Skipped     uint `json:"skipped"`
	Unreachable uint `json:"unreachable"`
}

type PlayOutput struct {
	CustomStats       map[string]any             `json:"custom_stats"`
	GlobalCustomStats map[string]any             `json:"global_custom_stats"`
	Results           []PlayOutputResult         `json:"plays"`
	Stats             map[string]PlayOutputStats `json:"stats"`
}

func GetTaskResult[T any](playOutput PlayOutput, play int, task int) (T, error) {
	var taskOutput T
	if play > len(playOutput.Results)-1 {
		return taskOutput, fmt.Errorf(
			"not enough results in play output to reach index '%d' (len=%d)",
			play,
			len(playOutput.Results),
		)
	}
	if task > len(playOutput.Results[play].Tasks)-1 {
		return taskOutput, fmt.Errorf(
			"not enough task results in play output to reach index '%d' (len=%d)",
			task,
			len(playOutput.Results[play].Tasks),
		)
	}
	var host string
	for h := range playOutput.Results[play].Tasks[task].Hosts {
		host = h
		break
	}
	taskOutputUntyped := playOutput.Results[play].Tasks[task].Hosts[host]
	data, err := json.Marshal(taskOutputUntyped)
	if err != nil {
		return taskOutput, err
	}
	err = json.Unmarshal(data, &taskOutput)
	return taskOutput, err
}

type Play struct {
	GatherFacts bool
	Become      bool
	Check       bool
	Tasks       []any
}

func RunPlay(ctx context.Context, connection *types.Connection, plays ...Play) (PlayOutput, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.RunPlay", trace.WithAttributes(
		attribute.String("exec.strategy", "ansible"),
		attribute.String("connection.host", *connection.Host),
	))
	defer span.End()

	playbook := []map[string]any{}
	for _, play := range plays {
		playbook = append(playbook, map[string]any{
			"hosts":        "all",
			"gather_facts": play.GatherFacts,
			"become":       play.Become,
			"diff":         true,
			"check_mode":   play.Check,
			"tasks":        play.Tasks,
		})
	}

	var playOutput PlayOutput

	playbookData, err := json.Marshal(playbook)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return playOutput, err
	}

	resultData, err := RunPlaybook(ctx, connection, playbookData)

	playOutputErr := json.Unmarshal(resultData, &playOutput)
	if playOutputErr != nil {
		err = errors.Join(err, playOutputErr)
	}
	if err == nil {
		span.SetStatus(codes.Ok, "")
	} else {
		span.SetStatus(codes.Error, err.Error())
	}
	return playOutput, err
}
