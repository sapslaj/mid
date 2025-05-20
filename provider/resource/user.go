package resource

import (
	"context"
	"errors"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
)

type User struct{}

type UserArgs struct {
	// TODO: support more features
	Name            *string              `pulumi:"name,optional"`
	Ensure          *string              `pulumi:"ensure,optional"`
	GroupsExclusive *bool                `pulumi:"groupsExclusive,optional"`
	Comment         *string              `pulumi:"comment,optional"`
	Local           *bool                `pulumi:"local,optional"`
	Group           *string              `pulumi:"group,optional"`
	Groups          *[]string            `pulumi:"groups,optional"`
	Home            *string              `pulumi:"home,optional"`
	Force           *bool                `pulumi:"force,optional"`
	ManageHome      *bool                `pulumi:"manageHome,optional"`
	NonUnique       *bool                `pulumi:"nonUnique,optional"`
	Password        *string              `pulumi:"password,optional"`
	Shell           *string              `pulumi:"shell,optional"`
	Skeleton        *string              `pulumi:"skeleton,optional"`
	System          *bool                `pulumi:"system,optional"`
	Uid             *int                 `pulumi:"uid,optional"`
	UidMax          *int                 `pulumi:"uidMax,optional"`
	UidMin          *int                 `pulumi:"uidMin,optional"`
	Umask           *string              `pulumi:"umask,optional"`
	UpdatePassword  *string              `pulumi:"updatePassword,optional"`
	Triggers        *types.TriggersInput `pulumi:"triggers,optional"`
}

type UserState struct {
	UserArgs
	Name     string               `pulumi:"name"`
	Triggers types.TriggersOutput `pulumi:"triggers"`
}

func (r User) argsToTaskParameters(input UserArgs) (ansible.UserParameters, error) {
	if input.Name == nil {
		return ansible.UserParameters{}, errors.New("someone forgot to set the auto-named input.Name")
	}
	groupsExclusive := false
	if input.GroupsExclusive != nil {
		groupsExclusive = *input.GroupsExclusive
	}
	return ansible.UserParameters{
		Name:           *input.Name,
		State:          input.Ensure,
		Append:         ptr.Of(!groupsExclusive),
		Comment:        input.Comment,
		Local:          input.Local,
		Group:          input.Group,
		Groups:         input.Groups,
		Home:           input.Home,
		Force:          input.Force,
		CreateHome:     input.ManageHome,
		MoveHome:       input.ManageHome,
		Remove:         input.ManageHome,
		NonUnique:      input.NonUnique,
		Password:       input.Password,
		Shell:          input.Shell,
		Skeleton:       input.Skeleton,
		System:         input.System,
		Uid:            input.Uid,
		UidMax:         input.UidMax,
		UidMin:         input.UidMin,
		Umask:          input.Umask,
		UpdatePassword: input.UpdatePassword,
	}, nil
}

func (r User) updateState(olds UserState, news UserArgs, changed bool) UserState {
	olds.UserArgs = news
	if news.Name != nil {
		olds.Name = *news.Name
	}
	olds.Triggers = types.UpdateTriggerState(olds.Triggers, news.Triggers, changed)
	return olds
}

func (r User) Diff(
	ctx context.Context,
	id string,
	olds UserState,
	news UserArgs,
) (p.DiffResponse, error) {
	diff := p.DiffResponse{
		HasChanges:          false,
		DetailedDiff:        map[string]p.PropertyDiff{},
		DeleteBeforeReplace: false,
	}

	if news.Name == nil {
		news.Name = &olds.Name
	} else if *news.Name != olds.Name {
		diff.HasChanges = true
		diff.DetailedDiff["name"] = p.PropertyDiff{
			Kind:      p.UpdateReplace,
			InputDiff: true,
		}
	}

	diff = types.MergeDiffResponses(
		diff,
		types.DiffAttributes(olds, news, []string{
			"ensure",
			"groupsExclusive",
			"comment",
			"local",
			"group",
			"groups",
			"home",
			"force",
			"manageHome",
			"nonUnique",
			"password",
			"shell",
			"skeleton",
			"system",
			"uid",
			"uidMax",
			"uidMin",
			"umask",
			"updatePassword",
		}),
		types.DiffTriggers(olds, news),
	)

	return diff, nil
}

func (r User) Create(
	ctx context.Context,
	name string,
	input UserArgs,
	preview bool,
) (string, UserState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if input.Name == nil {
		input.Name = ptr.Of(name)
	}

	state := r.updateState(UserState{}, input, true)

	id, err := resource.NewUniqueHex(name, 8, 0)
	if err != nil {
		return "", state, err
	}

	parameters, err := r.argsToTaskParameters(input)
	if err != nil {
		return id, state, err
	}

	call, err := parameters.ToRPCCall()
	if err != nil {
		return id, state, err
	}
	call.Args.Check = preview

	if preview {
		canConnect, _ := executor.CanConnect(ctx, config.Connection, 4)
		if !canConnect {
			return id, state, nil
		}
	}

	agent, err := executor.StartAgent(ctx, config.Connection)
	if err != nil {
		return id, state, err
	}
	defer agent.Disconnect()

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, agent, call)
	if err != nil || !callResult.Result.Success {
		return id, state, fmt.Errorf(
			"creating user failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
	}

	return id, state, nil
}

func (r User) Read(
	ctx context.Context,
	id string,
	inputs UserArgs,
	state UserState,
) (string, UserArgs, UserState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if inputs.Name == nil {
		inputs.Name = ptr.Of(state.Name)
	}

	parameters, err := r.argsToTaskParameters(inputs)
	if err != nil {
		return id, inputs, state, err
	}

	call, err := parameters.ToRPCCall()
	if err != nil {
		return id, inputs, state, err
	}
	call.Args.Check = true

	canConnect, err := executor.CanConnect(ctx, config.Connection, 4)

	if !canConnect {
		return id, inputs, UserState{
			UserArgs: inputs,
		}, nil
	}

	agent, err := executor.StartAgent(ctx, config.Connection)
	if err != nil {
		return id, inputs, state, err
	}
	defer agent.Disconnect()

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, agent, call)
	if err != nil || !callResult.Result.Success {
		return id, inputs, state, fmt.Errorf(
			"reading user failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
	}

	result, err := ansible.UserReturnFromRPCResult(callResult)
	if err != nil {
		return id, inputs, state, err
	}

	state = r.updateState(state, inputs, result.IsChanged())

	return id, inputs, state, nil
}

func (r User) Update(
	ctx context.Context,
	id string,
	olds UserState,
	news UserArgs,
	preview bool,
) (UserState, error) {
	config := infer.GetConfig[types.Config](ctx)

	if news.Name == nil {
		news.Name = ptr.Of(olds.Name)
	}

	parameters, err := r.argsToTaskParameters(news)
	if err != nil {
		return olds, err
	}

	call, err := parameters.ToRPCCall()
	if err != nil {
		return olds, err
	}
	call.Args.Check = preview

	if preview {
		call.Args.Check = true
		canConnect, _ := executor.CanConnect(ctx, config.Connection, 4)
		if !canConnect {
			return olds, nil
		}
	}

	agent, err := executor.StartAgent(ctx, config.Connection)
	if err != nil {
		return olds, err
	}
	defer agent.Disconnect()

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, agent, call)
	if err != nil || !callResult.Result.Success {
		return olds, fmt.Errorf(
			"updating user failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
	}

	result, err := ansible.UserReturnFromRPCResult(callResult)
	if err != nil {
		return olds, err
	}

	state := r.updateState(olds, news, result.IsChanged())

	return state, nil
}

func (r User) Delete(
	ctx context.Context,
	id string,
	props UserState,
) error {
	if props.Ensure != nil && *props.Ensure == "absent" {
		return nil
	}

	config := infer.GetConfig[types.Config](ctx)

	args := props.UserArgs
	args.Name = &props.Name

	parameters, err := r.argsToTaskParameters(args)
	if err != nil {
		return err
	}
	parameters.State = ptr.Of("absent")

	call, err := parameters.ToRPCCall()
	if err != nil {
		return err
	}

	canConnect, err := executor.CanConnect(ctx, config.Connection, 10)

	if !canConnect {
		if config.GetDeleteUnreachable() {
			return nil
		}

		if err == nil {
			return fmt.Errorf("cannot connect to host")
		} else {
			return fmt.Errorf("cannot connect to host: %w", err)
		}
	}

	agent, err := executor.StartAgent(ctx, config.Connection)
	if err != nil {
		return err
	}
	defer agent.Disconnect()

	callResult, err := executor.CallAgent[rpc.AnsibleExecuteArgs, rpc.AnsibleExecuteResult](ctx, agent, call)
	if err != nil || !callResult.Result.Success {
		return fmt.Errorf(
			"deleting user failed: stderr=%s stdout=%s, err=%w",
			callResult.Result.Stderr,
			callResult.Result.Stdout,
			err,
		)
	}

	return nil
}
