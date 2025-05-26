package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sapslaj/mid/pkg/telemetry"
	"github.com/sapslaj/mid/provider/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func RunPlaybook(ctx context.Context, connection *types.Connection, playbook []byte) ([]byte, error) {
	ctx, span := Tracer.Start(ctx, "mid/provider/executor.RunPlaybook", trace.WithAttributes(
		attribute.String("exec.strategy", "ansible"),
		attribute.String("connection.host", *connection.Host),
		attribute.String("ansible.playbook", string(playbook)),
	))
	defer span.End()
	logger := telemetry.LoggerFromContext(ctx).With(
		slog.String("connection.host", *connection.Host),
	)

	logger.DebugContext(ctx, "RunPlaybook: creating temp dir")
	dir, err := os.MkdirTemp("", "pulumi-mid")
	if err != nil {
		logger.ErrorContext(ctx, "RunPlaybook: error creating temp dir", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer os.RemoveAll(dir)
	logger.DebugContext(ctx, "RunPlaybook: created temp dir", slog.String("dir", dir))

	playbookPath := filepath.Join(dir, "play.yaml")
	logger.DebugContext(ctx, "RunPlaybook: writing playbook", slog.String("playbook_path", playbookPath))
	err = os.WriteFile(playbookPath, playbook, 0600)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"RunPlaybook: error writing playbook",
			slog.String("playbook_path", playbookPath),
			slog.Any("error", err),
		)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	logger.DebugContext(ctx, "RunPlaybook: wrote playbook playbook", slog.String("playbook_path", playbookPath))

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
		logger.DebugContext(ctx, "RunPlaybook: writing private key file", slog.String("private_key_path", privateKeyPath))
		err = os.WriteFile(privateKeyPath, []byte(*connection.PrivateKey), 0400)
		if err != nil {
			logger.ErrorContext(
				ctx,
				"RunPlaybook: error writing private key file",
				slog.String("private_key_path", privateKeyPath),
				slog.Any("error", err),
			)
			return nil, err
		}
		inventoryVars["ansible_ssh_private_key_file"] = privateKeyPath
		inventoryVars["ansible_ssh_common_args"] = inventoryVars["ansible_ssh_common_args"].(string) + " -o IdentitiesOnly=yes"
	}

	// TODO: proxy support
	// TODO: agentSocketPath support
	// TODO: dialErrorLimit support?
	// TODO: perDialTimeout support?
	// TODO: privateKeyPassword support?

	logger.DebugContext(ctx, "RunPlaybook: building inventory data")
	inventoryData, err := json.Marshal(map[string]any{
		"all": map[string]any{
			"hosts": map[string]any{
				*connection.Host: nil,
			},
			"vars": inventoryVars,
		},
	})
	if err != nil {
		logger.ErrorContext(
			ctx,
			"RunPlaybook: error building inventory data",
			slog.String("inventory_data", string(inventoryData)),
			slog.Any("error", err),
		)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.String("ansible.inventory", string(inventoryData)))
	logger.DebugContext(ctx, "RunPlaybook: built inventory data", slog.String("inventory_data", string(inventoryData)))

	inventoryPath := filepath.Join(dir, "inventory.yaml")
	logger.DebugContext(ctx, "RunPlaybook: writing inventory file", slog.String("inventory_path", inventoryPath))
	err = os.WriteFile(inventoryPath, inventoryData, 0600)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"RunPlaybook: error writing inventory file",
			slog.String("inventory_path", string(inventoryPath)),
			slog.Any("error", err),
		)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	logger.DebugContext(ctx, "RunPlaybook: wrote inventory file", slog.String("inventory_path", inventoryPath))

	logger.DebugContext(ctx, "RunPlaybook: building ansible-playbook command")
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

	logger.DebugContext(
		ctx,
		"running ansible-playbook command",
		telemetry.SlogJSON("cmd_args", cmd.Args),
		slog.String("cmd_dir", cmd.Dir),
	)

	err = cmd.Run()

	stderr := stderrBuffer.String()
	stdout := stdoutBuffer.String()
	exitCode := cmd.ProcessState.ExitCode()

	span.SetAttributes(
		attribute.Int("ansible.exit_code", exitCode),
		attribute.String("ansible.stderr", stderr),
		attribute.String("ansible.stdout", stdout),
	)

	if err != nil {
		err = fmt.Errorf(
			"command exited with non success code: %d stderr=%s stdout=%s err=%w",
			exitCode,
			stderr,
			stdout,
			err,
		)
		logger.ErrorContext(
			ctx,
			"RunPlaybook: ansible-playbook command failed",
			telemetry.SlogJSON("cmd_args", cmd.Args),
			slog.String("cmd_dir", cmd.Dir),
			slog.Int("cmd_exit_code", exitCode),
			slog.String("cmd_stderr", stderr),
			slog.String("cmd_stdout", stdout),
			slog.Any("error", err),
		)
		span.SetStatus(codes.Error, err.Error())
		return stdoutBuffer.Bytes(), err
	}

	logger.DebugContext(
		ctx,
		"RunPlaybook: ansible-playbook command succeeded",
		telemetry.SlogJSON("cmd_args", cmd.Args),
		slog.String("cmd_dir", cmd.Dir),
		slog.Int("cmd_exit_code", exitCode),
		slog.String("cmd_stderr", stderr),
		slog.String("cmd_stdout", stdout),
	)
	span.SetStatus(codes.Ok, "")
	return []byte(stdout), nil
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
	logger := telemetry.LoggerFromContext(ctx).With(
		slog.String("connection.host", *connection.Host),
	)

	var playOutput PlayOutput

	connectAttempts := 4
	for _, play := range plays {
		if !play.Check {
			connectAttempts = 10
		}
	}

	logger.DebugContext(ctx, "RunPlay: doing connection attempt", slog.Int("connection_attempts", connectAttempts))
	canConnect, err := CanConnect(ctx, connection, connectAttempts)
	if !canConnect || err != nil {
		if err == nil {
			err = errors.Join(ErrUnreachable, fmt.Errorf("cannot connect to host"))
		} else {
			err = errors.Join(ErrUnreachable, fmt.Errorf("cannot connect to host: %w", err))
		}
		span.SetStatus(codes.Error, err.Error())
		logger.ErrorContext(ctx, "RunPlay: host unreachable", slog.Any("error", err))
		return playOutput, err
	}

	logger.DebugContext(ctx, "RunPlay: building playbook")
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
	logger.DebugContext(ctx, "RunPlay: built playbook", telemetry.SlogJSON("playbook", playbook))

	playbookData, err := json.Marshal(playbook)
	if err != nil {
		logger.ErrorContext(ctx, "RunPlay: error marshalling playbook", slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
		return playOutput, err
	}

	logger.DebugContext(ctx, "RunPlay: running playbook")
	resultData, err := RunPlaybook(ctx, connection, playbookData)
	logger.DebugContext(ctx, "RunPlay: finished running playbook", slog.String("result_data", string(resultData)))

	playOutputErr := json.Unmarshal(resultData, &playOutput)
	if playOutputErr != nil {
		logger.ErrorContext(ctx, "RunPlay: error unmarshalling result", slog.Any("error", playOutputErr))
		err = errors.Join(err, playOutputErr)
	}
	if err == nil {
		span.SetStatus(codes.Ok, "")
		logger.DebugContext(ctx, "RunPlay: got result", telemetry.SlogJSON("result", playOutput))
	} else {
		logger.ErrorContext(ctx, "RunPlay: got result", telemetry.SlogJSON("result", playOutput), slog.Any("error", err))
		span.SetStatus(codes.Error, err.Error())
	}
	return playOutput, err
}
