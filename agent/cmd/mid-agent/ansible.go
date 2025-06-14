package main

import (
	"bytes"
	_ "embed"
	"os"
	"path"

	"github.com/sapslaj/mid/agent/untar"
)

//go:generate just agent-ansible-bundle
//go:embed ansible.tar.gz
var AnsibleTarball []byte

func InstallAnsible() error {
	targetDir := path.Join(".mid", "ansible")
	return untar.Untar(bytes.NewReader(AnsibleTarball), targetDir)
}

func UninstallAnsible() error {
	targetDir := path.Join(".mid", "ansible")
	return os.RemoveAll(targetDir)
}
