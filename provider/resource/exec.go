package resource

import (
	"context"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
	"github.com/sapslaj/mid/ptr"
)

type Exec struct{}

type ExecArgs struct {
	Create              types.ExecCommand    `pulumi:"create"`
	Update              *types.ExecCommand   `pulumi:"update,optional"`
	Delete              *types.ExecCommand   `pulumi:"delete,optional"`
	ExpandArgumentVars  *bool                `pulumi:"expandArgumentVars,optional"`
	DeleteBeforeReplace *bool                `pulumi:"deleteBeforeReplace,optional"`
	Dir                 *string              `pulumi:"dir,optional"`
	Environment         *map[string]string   `pulumi:"environment,optional"`
	Logging             *types.ExecLogging   `pulumi:"logging,optional"`
	Triggers            *types.TriggersInput `pulumi:"triggers,optional"`
}

type ExecState struct {
	ExecArgs
	Stdout   string               `pulumi:"stdout"`
	Stderr   string               `pulumi:"stderr"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

type commandTaskParameters struct {
	Argv               []string `json:"argv"`
	Chdir              *string  `json:"chdir,omitempty"`
	ExpandArgumentVars bool     `json:"expand_argument_vars"`
	Stdin              *string  `json:"stdin,omitempty"`
	StripEmptyEnds     *bool    `json:"strip_empty_ends,omitempty"`
}

type commandTaskResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

func (r Exec) canUseRPC(input ExecArgs) bool {
	if input.ExpandArgumentVars != nil {
		if *input.ExpandArgumentVars {
			return false
		}
	}
	return true
}

func (r Exec) argsToRPCCall(input ExecArgs, lifecycle string) (rpc.RPCCall[rpc.ExecArgs], error) {
	environment := map[string]string{}

	var execCommand types.ExecCommand
	switch lifecycle {
	case "create":
		execCommand = input.Create
	case "update":
		if input.Update != nil {
			execCommand = *input.Update
		} else {
			execCommand = input.Create
		}
	case "delete":
		if input.Delete == nil {
			return rpc.RPCCall[rpc.ExecArgs]{}, nil
		}
		execCommand = *input.Delete
	default:
		panic("unknown lifecycle: " + lifecycle)
	}

	chdir := ""
	if input.Dir != nil {
		chdir = *input.Dir
	}
	if execCommand.Dir != nil {
		chdir = *execCommand.Dir
	}

	if input.Environment != nil {
		for key, value := range *input.Environment {
			environment[key] = value
		}
	}
	if execCommand.Environment != nil {
		for key, value := range *execCommand.Environment {
			environment[key] = value
		}
	}

	stdin := []byte{}
	if execCommand.Stdin != nil {
		stdin = []byte(*execCommand.Stdin)
	}

	return rpc.RPCCall[rpc.ExecArgs]{
		RPCFunction: rpc.RPCExec,
		Args: rpc.ExecArgs{
			Command:     execCommand.Command,
			Dir:         chdir,
			Environment: environment,
			Stdin:       stdin,
		},
	}, nil
}

func (r Exec) argsToTaskParameters(input ExecArgs, lifecycle string) (commandTaskParameters, map[string]string, error) {
	environment := map[string]string{}

	var execCommand types.ExecCommand
	switch lifecycle {
	case "create":
		execCommand = input.Create
	case "update":
		if input.Update != nil {
			execCommand = *input.Update
		} else {
			execCommand = input.Create
		}
	case "delete":
		if input.Delete == nil {
			return commandTaskParameters{}, environment, nil
		}
		execCommand = *input.Delete
	default:
		panic("unknown lifecycle: " + lifecycle)
	}

	chdir := input.Dir
	if execCommand.Dir != nil {
		chdir = execCommand.Dir
	}
	expandArgumentVars := false
	if input.ExpandArgumentVars != nil {
		expandArgumentVars = *input.ExpandArgumentVars
	}

	if input.Environment != nil {
		for key, value := range *input.Environment {
			environment[key] = value
		}
	}
	if execCommand.Environment != nil {
		for key, value := range *execCommand.Environment {
			environment[key] = value
		}
	}

	return commandTaskParameters{
		Argv:               execCommand.Command,
		Chdir:              chdir,
		Stdin:              execCommand.Stdin,
		ExpandArgumentVars: expandArgumentVars,
		StripEmptyEnds:     ptr.Of(false),
	}, environment, nil
}

func (r Exec) updateState(olds ExecState, news ExecArgs, changed bool) ExecState {
	olds.ExecArgs = news
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r Exec) updateStateFromRPCResult(olds ExecState, news ExecArgs, result rpc.RPCResult[rpc.ExecResult]) ExecState {
	logging := types.ExecLoggingStdoutAndStderr
	if news.Logging != nil {
		logging = *news.Logging
	}
	switch logging {
	case types.ExecLoggingNone:
		olds.Stderr = ""
		olds.Stdout = ""
	case types.ExecLoggingStderr:
		olds.Stderr = string(result.Result.Stderr)
		olds.Stdout = ""
	case types.ExecLoggingStdout:
		olds.Stderr = ""
		olds.Stdout = string(result.Result.Stdout)
	case types.ExecLoggingStdoutAndStderr:
		olds.Stderr = string(result.Result.Stderr)
		olds.Stdout = string(result.Result.Stdout)
	default:
		panic("unknown logging: " + logging)
	}
	return olds
}

func (r Exec) updateStateFromOutput(olds ExecState, news ExecArgs, output commandTaskResult) ExecState {
	logging := types.ExecLoggingStdoutAndStderr
	if news.Logging != nil {
		logging = *news.Logging
	}
	switch logging {
	case types.ExecLoggingNone:
		olds.Stderr = ""
		olds.Stdout = ""
	case types.ExecLoggingStderr:
		olds.Stderr = output.Stderr
		olds.Stdout = ""
	case types.ExecLoggingStdout:
		olds.Stderr = ""
		olds.Stdout = output.Stdout
	case types.ExecLoggingStdoutAndStderr:
		olds.Stderr = output.Stderr
		olds.Stdout = output.Stdout
	default:
		panic("unknown logging: " + logging)
	}
	return olds
}

func (r Exec) Diff(
	ctx context.Context,
	id string,
	olds ExecState,
	news ExecArgs,
) (p.DiffResponse, error) {
	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	if news.DeleteBeforeReplace != nil {
		diff.DeleteBeforeReplace = *news.DeleteBeforeReplace
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, []string{
			"create",
			"update",
			"delete",
			"expandArgumentVars",
			"dir",
			"environment",
			"logging",
		}),
		types.DiffTriggers(olds, news),
	)

	return diff, nil
}

func (r Exec) Create(
	ctx context.Context,
	name string,
	input ExecArgs,
	preview bool,
) (string, ExecState, error) {
	config := infer.GetConfig[types.Config](ctx)

	state := r.updateState(ExecState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	if r.canUseRPC(input) {
		agent, err := executor.StartAgent(ctx, config.Connection)
		if err != nil {
			return id, state, err
		}
		defer agent.Disconnect()

		call, err := r.argsToRPCCall(input, "create")
		if err != nil {
			return id, state, err
		}

		if !preview {
			result, err := executor.CallAgent[rpc.ExecArgs, rpc.ExecResult](agent, call)
			if err != nil {
				return id, state, err
			}
			state = r.updateStateFromRPCResult(state, input, result)
		}
	} else {
		parameters, environment, err := r.argsToTaskParameters(input, "create")
		if err != nil {
			return id, state, err
		}

		if !preview {
			output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
				GatherFacts: false,
				Become:      true,
				Check:       false,
				Tasks: []any{
					map[string]any{
						"ansible.builtin.command": parameters,
						"environment":             environment,
					},
				},
			})
			if err != nil {
				return id, state, err
			}
			result, err := executor.GetTaskResult[commandTaskResult](output, 0, 0)
			if err != nil {
				return id, state, err
			}
			state = r.updateStateFromOutput(state, input, result)
		}
	}

	return id, state, nil
}

// func (r Exec) Read(
// 	ctx context.Context,
// 	id string,
// 	inputs ExecArgs,
// 	state ExecState,
// ) (string, ExecArgs, ExecState, error) {
// 	config := infer.GetConfig[types.Config](ctx)
// }

func (r Exec) Update(
	ctx context.Context,
	id string,
	olds ExecState,
	news ExecArgs,
	preview bool,
) (ExecState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if r.canUseRPC(news) {
		agent, err := executor.StartAgent(ctx, config.Connection)
		if err != nil {
			return olds, err
		}
		defer agent.Disconnect()

		call, err := r.argsToRPCCall(news, "update")
		if err != nil {
			return olds, err
		}

		if !preview {
			result, err := executor.CallAgent[rpc.ExecArgs, rpc.ExecResult](agent, call)
			if err != nil {
				return olds, err
			}
			olds = r.updateStateFromRPCResult(olds, news, result)
		}
	} else {
		parameters, environment, err := r.argsToTaskParameters(news, "update")
		if err != nil {
			return olds, err
		}

		if !preview {
			output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
				GatherFacts: false,
				Become:      true,
				Check:       false,
				Tasks: []any{
					map[string]any{
						"ansible.builtin.command": parameters,
						"environment":             environment,
					},
				},
			})
			result, err := executor.GetTaskResult[commandTaskResult](output, 0, 0)
			if err != nil {
				return olds, err
			}
			olds = r.updateStateFromOutput(olds, news, result)
		}
	}
	state := r.updateState(olds, news, true)

	return state, nil
}

func (r Exec) Delete(ctx context.Context, id string, props ExecState) error {
	if props.Delete == nil {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	if r.canUseRPC(props.ExecArgs) {
		agent, err := executor.StartAgent(ctx, config.Connection)
		if err != nil {
			return err
		}
		defer agent.Disconnect()

		call, err := r.argsToRPCCall(props.ExecArgs, "delete")
		if err != nil {
			return err
		}

		_, err = executor.CallAgent[rpc.ExecArgs, rpc.ExecResult](agent, call)
		if err != nil {
			return err
		}
	} else {
		parameters, environment, err := r.argsToTaskParameters(props.ExecArgs, "delete")
		if err != nil {
			return err
		}

		_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
			GatherFacts: false,
			Become:      true,
			Check:       false,
			Tasks: []any{
				map[string]any{
					"ansible.builtin.command": parameters,
					"environment":             environment,
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
