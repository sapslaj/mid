package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

//go:generate just agent-ansible-bundle
//go:embed ansible.tar.gz
var AnsibleTarball []byte

func InstallAnsible() error {
	targetDir := path.Join(".mid", "ansible")
	err := os.MkdirAll(targetDir, 0o700)
	if err != nil {
		return err
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(AnsibleTarball))
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		targetFilepath := path.Join(targetDir, header.Name)

		switch header.Typeflag {
		case tar.TypeRegA:
			fallthrough
		case tar.TypeReg:
			f, err := os.OpenFile(targetFilepath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tarReader)
			if err != nil {
				return err
			}
			f.Close()
		case tar.TypeDir:
			fileinfo, err := os.Stat(targetFilepath)
			if err == nil && !fileinfo.IsDir() {
				err = os.RemoveAll(targetFilepath)
				if err != nil {
					return err
				}
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			err = os.MkdirAll(targetFilepath, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		case tar.TypeLink:
			fallthrough
		case tar.TypeSymlink:
			fallthrough
		case tar.TypeChar:
			fallthrough
		case tar.TypeBlock:
			fallthrough
		case tar.TypeFifo:
			fallthrough
		case tar.TypeCont:
			fallthrough
		case tar.TypeXHeader:
			fallthrough
		case tar.TypeXGlobalHeader:
			fallthrough
		case tar.TypeGNUSparse:
			fallthrough
		case tar.TypeGNULongName:
			fallthrough
		case tar.TypeGNULongLink:
			fallthrough
		default:
			return fmt.Errorf("unsupported tar header: '%v'", header.Typeflag)
		}
	}

	return nil
}

func UninstallAnsible() error {
	targetDir := path.Join(".mid", "ansible")
	return os.RemoveAll(targetDir)
}
