package testmachine

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anatol/vmtest"
	"github.com/ory/dockertest/v3"
	"github.com/sapslaj/mid/pkg/env"
	"golang.org/x/crypto/ssh"
)

type Backend string

const (
	DockerBackend Backend = "docker"
	QEMUBackend   Backend = "qemu"
)

type Config struct {
	Name        string
	Backend     Backend
	DataPath    string
	SSHUsername string
}

type TestMachine struct {
	// common stuffs
	Config      Config
	SSHUsername string
	SSHPassword string
	SSHHost     string
	SSHPort     int
	SSHAddress  string
	SSHClient   *ssh.Client
	DataPath    string

	// Docker
	DockertestPool  *dockertest.Pool
	DockerContainer *dockertest.Resource

	// QEMU
	QemuInstance *vmtest.Qemu
}

func (tm *TestMachine) Close() error {
	if tm.DockerContainer != nil {
		tm.DockertestPool.Purge(tm.DockerContainer)
	}
	if tm.QemuInstance != nil {
		tm.QemuInstance.Kill()
	}
	return nil
}

func SafeName(s string) string {
	re := regexp.MustCompile(`['\"!@#$%^&\*\(\)\[\]\{\};:\,\./<>\?\|` + "`" + `~=_+ ]`)
	return re.ReplaceAllString(strings.ToLower(s), "-")
}

func New(t *testing.T, config Config) (*TestMachine, error) {
	t.Helper()

	if config.Name == "" {
		config.Name = "mid-" + SafeName(t.Name())
	}
	t.Logf("testmachine: using name %q", config.Name)

	if config.Backend == "" {
		config.Backend = DockerBackend
	}
	t.Logf("testmachine: using backend %q", config.Backend)

	if config.SSHUsername == "" {
		config.SSHUsername = "ubuntu"
	}
	t.Logf("testmachine: using ssh username %q", config.SSHUsername)

	if config.DataPath == "" {
		// HACK: gotta do some nasty stuff to get the path to where all of the data
		// files are
		b, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return nil, err
		}
		config.DataPath = path.Join(strings.TrimSpace(string(b)), "tests", "testmachine")
	}
	t.Logf("testmachine: using datapath %q", config.DataPath)

	var tm *TestMachine
	var err error
	switch config.Backend {
	case DockerBackend:
		tm, err = NewDocker(t, config)
	case QEMUBackend:
		tm, err = NewQEMU(t, config)
	default:
		return nil, fmt.Errorf("unknown backend: %q", config.Backend)
	}
	tm.SSHUsername = config.SSHUsername
	tm.SSHPassword = "hunter2" // NOTE: hardcoded password
	if tm.SSHAddress == "" {
		tm.SSHAddress = fmt.Sprintf("%s:%d", tm.SSHHost, tm.SSHPort)
	}
	if err != nil {
		return tm, err
	}

	for attempt := 1; attempt <= 10; attempt++ {
		t.Logf("(attempt %d/10) connecting to test machine at address %s over SSH", attempt, tm.SSHAddress)
		tm.SSHClient, err = ssh.Dial(
			"tcp",
			tm.SSHAddress,
			&ssh.ClientConfig{
				User:            config.SSHUsername,
				Auth:            []ssh.AuthMethod{ssh.Password(tm.SSHPassword)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
		)
		if attempt == 10 || err == nil {
			break
		}
		wait := time.Duration(attempt) * 5 * time.Second
		t.Logf("(attempt %d/10) error connecting to test machine: %v", attempt, err)
		t.Logf("(attempt %d/10) trying again in %s", attempt, wait)
		time.Sleep(wait)
	}
	if err != nil {
		return tm, err
	}

	return tm, nil
}

func NewDocker(t *testing.T, config Config) (*TestMachine, error) {
	t.Helper()

	t.Logf("launching Docker test machine")
	tm := &TestMachine{
		Config: config,
	}

	var err error
	tm.DockertestPool, err = dockertest.NewPool("")
	if err != nil {
		return tm, err
	}

	existing, exists := tm.DockertestPool.ContainerByName(config.Name)
	if exists {
		t.Logf("removing orphaned container")
		err = tm.DockertestPool.Purge(existing)
		if err != nil {
			return tm, err
		}
	}

	tm.DockerContainer, err = tm.DockertestPool.BuildAndRun(
		config.Name,
		path.Join(config.DataPath, "Dockerfile"),
		[]string{},
	)
	if err != nil {
		return tm, err
	}

	tm.SSHPort, err = strconv.Atoi(tm.DockerContainer.GetPort("22/tcp"))
	if err != nil {
		return tm, err
	}

	tm.SSHHost = tm.DockerContainer.GetBoundIP("22/tcp")

	return tm, nil
}

var QEMUDownloadMutex = sync.Mutex{}

func NewQEMU(t *testing.T, config Config) (*TestMachine, error) {
	t.Helper()

	t.Logf("launching QEMU test machine")
	tm := &TestMachine{
		Config: config,
	}

	img := path.Join(config.DataPath, "noble-server-cloudimg-amd64.img")

	err := DownloadQEMUImage(
		t,
		"http://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
		img,
	)
	if err != nil {
		return tm, err
	}

	tm.SSHPort = rand.Intn(32768) + 1024
	tm.SSHHost = "localhost"

	buildQemuParams := func(kvm bool) []string {
		params := []string{
			"-snapshot",
			"-smp", "2", // XXX: does this need to be configurable?
			"-m", "1024", // XXX: does this need to be configurable?
			"-netdev", fmt.Sprintf("id=net00,type=user,hostfwd=tcp::%d-:22", tm.SSHPort),
			"-device", "virtio-net-pci,netdev=net00",
			"-drive", fmt.Sprintf("if=virtio,format=qcow2,file=%s", img),
			"-drive", fmt.Sprintf("if=virtio,format=raw,file=%s", path.Join(config.DataPath, "seed.img")),
		}
		if kvm {
			params = append(params, "-accel", "kvm")
		}
		return params
	}

	enableKvm, err := env.Get[bool]("ENABLE_KVM")
	if err != nil {
		if env.IsErrVarNotFound(err) {
			enableKvm = false
			if _, err := os.Stat("/dev/kvm"); err == nil {
				t.Log("KVM support detected, attempting to launch QEMU with KVM")
				enableKvm = true
			}
		} else {
			return tm, err
		}
	}

	qemuOptions := &vmtest.QemuOptions{
		OperatingSystem: vmtest.OS_LINUX,
		Architecture:    vmtest.QEMU_X86_64,
		Verbose:         false,
		Params:          buildQemuParams(enableKvm),
		Timeout:         10 * time.Minute,
	}

	tm.QemuInstance, err = vmtest.NewQemu(qemuOptions)
	if err != nil {
		_, isExitError := err.(*exec.ExitError)
		if isExitError && enableKvm {
			t.Log("failed to launch VM with KVM support, trying without KVM")
			qemuOptions.Params = buildQemuParams(false)
			tm.QemuInstance, err = vmtest.NewQemu(qemuOptions)
			if err != nil {
				return tm, err
			}
		} else {
			return tm, err
		}
	}

	t.Logf("waiting for cloud-init...")
	err = tm.QemuInstance.ConsoleExpect("running 'modules:final'")
	if err != nil {
		return tm, err
	}

	return tm, nil
}

func DownloadQEMUImage(t *testing.T, url string, dest string) error {
	t.Helper()

	QEMUDownloadMutex.Lock()
	defer QEMUDownloadMutex.Unlock()

	_, err := os.Stat(dest)
	if err == nil {
		t.Logf("%s already downloaded", dest)
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	t.Logf("downloading %s to %s", url, dest)
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	t.Logf("finished downloading %s", dest)

	return nil
}
