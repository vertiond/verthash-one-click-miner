package miners

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vertcoin-project/one-click-miner-vnext/logging"
	"github.com/vertcoin-project/one-click-miner-vnext/prerequisites"
	"github.com/vertcoin-project/one-click-miner-vnext/util"
)

type MinerBinary struct {
	Platform           string   `json:"platform"`
	GpuPlatformString  string   `json:"gpuplatform"`
	Url                string   `json:"url"`
	Prerequisites      []string `json:"minerPreRequisites"`
	Hash               string   `json:"sha256"`
	MainExecutableName string   `json:"mainExecutableName"`
	MultiGPUMiner      bool     `json:"multiGPUMiner"`
	EnableIntegrated   bool     `json:"enableIntegrated"`
	GPUType            util.GPUType
}

func GetMinerBinaries() []MinerBinary {
	binaries := []MinerBinary{}
	err := util.GetJson("https://raw.githubusercontent.com/vertiond/verthash-one-click-miner/master/miners.json", &binaries)
	if err != nil {
		logging.Errorf("Error fetching miner binaries: %v", err)
		return binaries
	}
	for i := range binaries {
		if binaries[i].GpuPlatformString == "AMD" {
			binaries[i].GPUType = util.GPUTypeAMD
		} else if binaries[i].GpuPlatformString == "NVIDIA" {
			binaries[i].GPUType = util.GPUTypeNVidia
		} else if binaries[i].GpuPlatformString == "INTEL" {
			binaries[i].GPUType = util.GPUTypeIntel
		} else {
			logging.Warnf("Found unrecognized platform [%s] in miners.json\n", binaries[i].GpuPlatformString)
		}
	}
	return binaries
}

type MinerImpl interface {
	ParseOutput(line string)
	Configure(args BinaryArguments) error
	HashRate() uint64
	ConstructCommandlineArgs(args BinaryArguments) []string
	AvailableGPUs() int8
}

func NewBinaryRunner(m MinerBinary) (*BinaryRunner, error) {
	br := &BinaryRunner{MinerBinary: m, lastStarted: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)}
	if !strings.HasPrefix(m.MainExecutableName, "VerthashMiner") {
		return nil, fmt.Errorf("Could not determine implementation for miner binary")
	}
	br.MinerImpl = NewVerthashMinerImpl(br)
	return br, nil
}

type BinaryArguments struct {
	StratumUrl       string
	StratumUsername  string
	StratumPassword  string
	EnableIntegrated bool
}

type BinaryRunner struct {
	MinerBinary      MinerBinary
	MinerImpl        MinerImpl
	cmd              *exec.Cmd
	Debug            bool
	running          bool
	lastStarted      time.Time
	rapidFails       int
	usedArgs         BinaryArguments
	freshlyInstalled bool // Install just ran (in the checks) and Start has not used it yet
}

func (b *BinaryRunner) logPrefix() string {
	return fmt.Sprintf("[Miner %s/%d]", b.MinerBinary.Platform, b.MinerBinary.GPUType)
}

func (b *BinaryRunner) Stop() error {

	if b.cmd == nil {
		// not started (yet)
		return nil
	}
	// Windows doesn't support Interrupt
	if runtime.GOOS == "windows" {
		_ = b.cmd.Process.Signal(os.Kill)
		return nil
	}

	go func() {
		time.Sleep(15 * time.Second)
		_ = b.cmd.Process.Signal(os.Kill)
	}()
	err := b.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return err
	}

	b.WaitUntilStopped()
	return nil
}

func (b *BinaryRunner) WaitUntilStopped() {
	stopped := make(chan bool)
	go func() {
		for {
			if !b.running {
				stopped <- true
				break
			}
			<-time.After(time.Second)
		}
	}()
	<-stopped
}

func (b *BinaryRunner) IsRunning() bool {
	return b.running
}

func (b *BinaryRunner) Install() error {
	// Check prerequisites
	for _, p := range b.MinerBinary.Prerequisites {
		err := prerequisites.Install(p)
		if err != nil {
			return err
		}
	}

	// Check if the archive is available and it has the right SHA sum. Download if not
	err := b.ensureAvailable()
	if err != nil {
		return err
	}

	// Always bring the unpacked files back in line with the archive, so no one
	// tampered with them on disk.
	err = b.unpack()
	if err != nil {
		return err
	}

	b.freshlyInstalled = true
	return nil
}

// ensureInstalled installs the miner unless the checks just did, so the start
// that follows them does not unpack the same archive a second time. Any later
// start installs afresh.
func (b *BinaryRunner) ensureInstalled() error {
	if b.freshlyInstalled {
		b.freshlyInstalled = false
		return nil
	}
	return b.Install()
}

type RunningState int

const (
	RunningStateRunning    RunningState = 0
	RunningStateRestarting RunningState = 1
	RunningStateRapidFail  RunningState = 2
)

func (b *BinaryRunner) CheckRunning() RunningState {
	if !b.IsRunning() {
		logging.Infof("Miner [%s] stopped running.", b.MinerBinary.MainExecutableName)
		if time.Since(b.lastStarted).Seconds() < 10 {
			// Rapid fail
			b.rapidFails++
			if b.rapidFails > 3 {
				logging.Infof("Miner [%s] is rapidly failing, not restarting it.", b.MinerBinary.MainExecutableName)
				return RunningStateRapidFail
			}
		}

		logging.Infof("Restarting miner [%s]", b.MinerBinary.MainExecutableName)
		err := b.restart()
		if err != nil {
			logging.Errorf("Error restarting miner: %v", err)
		}
		return RunningStateRestarting
	}
	return RunningStateRunning
}

func (b *BinaryRunner) HashRate() uint64 {
	return b.MinerImpl.HashRate()
}

func (b *BinaryRunner) restart() error {
	b.lastStarted = time.Now()
	return b.launch(b.MinerImpl.ConstructCommandlineArgs(b.usedArgs), true)
}

func (b *BinaryRunner) Start(args BinaryArguments) error {
	err := b.ensureInstalled()
	if err != nil {
		return err
	}

	b.usedArgs = args
	b.lastStarted = time.Now()
	b.rapidFails = 0

	// Check if there is a compatible GPU!
	if b.MinerImpl.AvailableGPUs() == 0 {
		return fmt.Errorf("err_no_gpus")
	}

	// The executable was freshly unpacked, by the checks or by ensureInstalled,
	// to ensure there's been no funny business. EnsureAvailable already checked
	// the SHA hash.
	err = b.launch(b.MinerImpl.ConstructCommandlineArgs(args), true)
	if err != nil {
		return err
	}

	return nil
}

func (b *BinaryRunner) unpackDir() string {
	return filepath.Join(util.DataDirectory(), "miners", fmt.Sprintf("unpacked-%s", b.MinerBinary.Hash))
}

func (b *BinaryRunner) downloadPath() string {
	return filepath.Join(util.DataDirectory(), "miners", b.MinerBinary.Hash)
}

func (b *BinaryRunner) launch(params []string, wait bool) error {
	exePath := b.findExecutable()
	if exePath == "" {
		return fmt.Errorf("Cannot find main miner binary in unpack folder")
	}
	logging.Debugf("Launching %s %v\n", exePath, params)
	b.cmd = exec.Command(exePath, params...)
	util.PrepareBackgroundCommand(b.cmd)
	b.cmd.Dir = filepath.Dir(exePath)
	r, w := io.Pipe()
	go func(b *BinaryRunner, rd io.Reader) {
		br := bufio.NewReader(rd)

		for {
			l, _, e := br.ReadLine()
			if e != nil {
				logging.Debugf("%sError on readline from stdout/err: %s", b.logPrefix(), e.Error())
				return
			}
			b.MinerImpl.ParseOutput(string(l))
		}
	}(b, r)
	b.cmd.Stderr = w
	b.cmd.Stdout = w
	err := b.cmd.Start()
	if err != nil {
		return err
	}
	if wait {
		b.running = true
		go func() {
			err := b.cmd.Wait()
			if err != nil {
				logging.Errorf("Error in miner: %v", err)
			}
			b.running = false
		}()
	}
	return nil
}

func (b *BinaryRunner) unpack() error {
	return UnpackArchive(b.MinerBinary.Url, b.downloadPath(), b.unpackDir())
}

func (b *BinaryRunner) findExecutable() string {
	mainExecutablePath := ""
	err := filepath.Walk(b.unpackDir(),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == b.MinerBinary.MainExecutableName {
				mainExecutablePath = path
			}
			return nil
		})
	if err != nil {
		logging.Errorf("Error finding executable: %v", err)
	}
	return mainExecutablePath
}

func (b *BinaryRunner) ensureAvailable() error {
	_ = os.Mkdir(filepath.Join(util.DataDirectory(), "miners"), 0700)
	return EnsureArchive(b.MinerBinary.Url, b.MinerBinary.Hash, b.downloadPath())
}
