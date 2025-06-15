package rpc

import (
	"strings"
)

type SystemdUnitShortStatusArgs struct {
	Name string
}

type SystemdUnitShortStatusResult struct {
	Name        string
	Exists      bool
	LoadState   string
	ActiveState string
	SubState    string
}

func SystemdUnitShortStatus(args SystemdUnitShortStatusArgs) (SystemdUnitShortStatusResult, error) {
	result := SystemdUnitShortStatusResult{
		Name: args.Name,
	}

	catResult, err := Exec(ExecArgs{
		Command: []string{
			"systemctl",
			"cat",
			args.Name,
		},
	})
	if err != nil {
		return result, err
	}

	result.Exists = catResult.ExitCode == 0

	if !result.Exists {
		return result, nil
	}

	listUnits, err := Exec(ExecArgs{
		Command: []string{
			"systemctl",
			"list-units",
			"--no-pager",
			"--plain",
			"--no-legend",
		},
	})
	if err != nil {
		return result, err
	}

	for line := range strings.Lines(string(listUnits.Stdout)) {
		fields := strings.Fields(line)
		if args.Name != fields[0] {
			continue
		}
		result.Exists = true
		result.LoadState = fields[1]
		result.ActiveState = fields[2]
		result.SubState = fields[3]
		return result, nil
	}

	return result, nil
}
