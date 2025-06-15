package rpc

import (
	"fmt"
	"os"

	"github.com/sapslaj/mid/pkg/cast"
)

type RPCFunction string

const (
	RPCAgentPing              RPCFunction = "AgentPing"
	RPCAnsibleExecute         RPCFunction = "AnsibleExecute"
	RPCClose                  RPCFunction = "Close"
	RPCExec                   RPCFunction = "Exec"
	RPCFileStat               RPCFunction = "FileStat"
	RPCSystemdUnitShortStatus RPCFunction = "SystemdUnitShortStatus"
	RPCUntar                  RPCFunction = "Untar"
)

type RPCCall[T any] struct {
	UUID        string
	RPCFunction RPCFunction
	Args        T
}

type RPCResult[T any] struct {
	UUID        string
	RPCFunction RPCFunction
	Result      T
	Error       string
}

func ServerRoute(rpcFunction RPCFunction, args any) (any, error) {
	switch rpcFunction {
	case RPCClose:
		os.Exit(0)
	case RPCAgentPing:
		var targs AgentPingArgs
		targs, err := cast.AnyToJSONT[AgentPingArgs](args)
		if err != nil {
			return nil, err
		}
		return AgentPing(targs)
	case RPCAnsibleExecute:
		var targs AnsibleExecuteArgs
		targs, err := cast.AnyToJSONT[AnsibleExecuteArgs](args)
		if err != nil {
			return nil, err
		}
		return AnsibleExecute(targs)
	case RPCExec:
		var targs ExecArgs
		targs, err := cast.AnyToJSONT[ExecArgs](args)
		if err != nil {
			return nil, err
		}
		return Exec(targs)
	case RPCFileStat:
		var targs FileStatArgs
		targs, err := cast.AnyToJSONT[FileStatArgs](args)
		if err != nil {
			return nil, err
		}
		return FileStat(targs)
	case RPCSystemdUnitShortStatus:
		var targs SystemdUnitShortStatusArgs
		targs, err := cast.AnyToJSONT[SystemdUnitShortStatusArgs](args)
		if err != nil {
			return nil, err
		}
		return SystemdUnitShortStatus(targs)
	case RPCUntar:
		var targs UntarArgs
		targs, err := cast.AnyToJSONT[UntarArgs](args)
		if err != nil {
			return nil, err
		}
		return Untar(targs)
	}

	return nil, fmt.Errorf("unsupported RPCFunction: %s", rpcFunction)
}
