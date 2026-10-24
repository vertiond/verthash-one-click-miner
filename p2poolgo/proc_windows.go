//go:build windows

package p2poolgo

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// prepareNodeCommand runs the console node without allocating a console.
func prepareNodeCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
}

var (
	jobOnce   sync.Once
	jobHandle windows.Handle
	jobErr    error
)

// nodeJob is a Job Object that kills its processes when its last handle
// closes. It is created once and its handle is never closed, so the node
// dies with OCM's process.
func nodeJob() (windows.Handle, error) {
	jobOnce.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			jobErr = fmt.Errorf("CreateJobObject: %w", err)
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}
		if _, err := windows.SetInformationJobObject(h, windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
			windows.CloseHandle(h)
			jobErr = fmt.Errorf("SetInformationJobObject: %w", err)
			return
		}
		jobHandle = h
	})
	return jobHandle, jobErr
}

// afterNodeStart assigns the started node to the kill-on-close job.
func afterNodeStart(cmd *exec.Cmd) error {
	job, err := nodeJob()
	if err != nil {
		return err
	}
	p, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("OpenProcess: %w", err)
	}
	defer windows.CloseHandle(p)
	if err := windows.AssignProcessToJobObject(job, p); err != nil {
		return fmt.Errorf("AssignProcessToJobObject: %w", err)
	}
	return nil
}
