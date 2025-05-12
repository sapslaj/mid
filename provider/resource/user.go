package resource

import (
	"context"
	"errors"
	"fmt"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/go/common/resource"

	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/provider/types"
	"github.com/sapslaj/mid/ptr"
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

type userTaskParameters struct {
	Append                       *bool     `json:"append,omitempty"`
	Authorization                *string   `json:"authorization,omitempty"`
	Comment                      *string   `json:"comment,omitempty"`
	CreateHome                   *bool     `json:"create_home,omitempty"`
	Expires                      *float64  `json:"expires,omitempty"`
	Force                        *bool     `json:"force,omitempty"`
	GenerateSSHKey               *bool     `json:"generate_ssh_key,omitempty"`
	Group                        *string   `json:"group,omitempty"`
	Groups                       *[]string `json:"groups,omitempty"`
	Hidden                       *bool     `json:"hidden,omitempty"`
	Home                         *string   `json:"home,omitempty"`
	Local                        *bool     `json:"local,omitempty"`
	LoginClass                   *string   `json:"login_class,omitempty"`
	MoveHome                     *bool     `json:"move_home,omitempty"`
	Name                         string    `json:"name"`
	NonUnique                    *bool     `json:"non_unique,omitempty"`
	Password                     *string   `json:"password,omitempty"`
	PasswordExpireAccountDisable *int      `json:"password_expire_account_disable,omitempty"`
	PasswordExpireMax            *int      `json:"password_expire_max,omitempty"`
	PasswordExpireMin            *int      `json:"password_expire_min,omitempty"`
	PasswordExpireWarn           *int      `json:"password_expire_warn,omitempty"`
	PasswordLock                 *bool     `json:"password_lock,omitempty"`
	Profile                      *string   `json:"profile,omitempty"`
	Remove                       *bool     `json:"remove,omitempty"`
	Role                         *string   `json:"role,omitempty"`
	Seuser                       *string   `json:"seuser,omitempty"`
	Shell                        *string   `json:"shell,omitempty"`
	Skeleton                     *string   `json:"skeleton,omitempty"`
	SSHKeyBits                   *int      `json:"ssh_key_bits,omitempty"`
	SSHKeyComment                *string   `json:"ssh_key_comment,omitempty"`
	SSHKeyFile                   *string   `json:"ssh_key_file,omitempty"`
	SSHKeyPassphrase             *string   `json:"ssh_key_passphrase,omitempty"`
	SSHKeyType                   *string   `json:"ssh_key_type,omitempty"`
	State                        *string   `json:"state,omitempty"`
	System                       *bool     `json:"system,omitempty"`
	Uid                          *int      `json:"uid,omitempty"`
	UidMax                       *int      `json:"uid_max,omitempty"`
	UidMin                       *int      `json:"uid_min,omitempty"`
	Umask                        *string   `json:"umask,omitempty"`
	UpdatePassword               *string   `json:"update_password,omitempty"`
}

type userTaskResult struct {
	Changed *bool `json:"changed,omitempty"`
	Diff    *any  `json:"diff,omitempty"`
}

func (result *userTaskResult) IsChanged() bool {
	changed := result.Changed != nil && *result.Changed
	hasDiff := result.Diff != nil
	return changed || hasDiff
}

func (r User) argsToTaskParameters(input UserArgs) (userTaskParameters, error) {
	if input.Name == nil {
		return userTaskParameters{}, errors.New("someone forgot to set the auto-named input.Name")
	}
	groupsExclusive := false
	if input.GroupsExclusive != nil {
		groupsExclusive = *input.GroupsExclusive
	}
	return userTaskParameters{
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

	canConnect, err := executor.CanConnect(ctx, config.Connection)

	if !canConnect {
		if preview {
			return id, state, nil
		}

		if err == nil {
			return id, state, fmt.Errorf("cannot connect to host")
		} else {
			return id, state, fmt.Errorf("cannot connect to host: %w", err)
		}
	}

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.user": parameters,
			},
		},
	})
	if err != nil {
		return id, state, err
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

	canConnect, err := executor.CanConnect(ctx, config.Connection)

	if !canConnect {
		return id, inputs, UserState{
			UserArgs: inputs,
		}, nil
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       true,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.user": parameters,
			},
		},
	})
	if err != nil {
		return id, inputs, state, err
	}

	result, err := executor.GetTaskResult[*serviceTaskResult](output, 0, 0)
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

	canConnect, err := executor.CanConnect(ctx, config.Connection)

	if !canConnect {
		if preview {
			return olds, nil
		}

		if err == nil {
			return olds, fmt.Errorf("cannot connect to host")
		} else {
			return olds, fmt.Errorf("cannot connect to host: %w", err)
		}
	}

	output, err := executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       preview,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.user": parameters,
			},
		},
	})
	if err != nil {
		return olds, err
	}

	result, err := executor.GetTaskResult[*serviceTaskResult](output, 0, 0)

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

	canConnect, err := executor.CanConnect(ctx, config.Connection)

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

	_, err = executor.RunPlay(ctx, config.Connection, executor.Play{
		GatherFacts: true,
		Become:      true,
		Check:       false,
		Tasks: []any{
			map[string]any{
				"ansible.builtin.user": parameters,
			},
		},
	})
	return err
}
