package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"unsafe"

	"github.com/elastic/go-seccomp-bpf/arch"
	"golang.org/x/sys/unix"
)

func run() error {
	var (
		cmd      *exec.Cmd
		ws       unix.WaitStatus
		pid      int
		sigInfo  SigInfoSys
		archInfo *arch.Info
		errno    unix.Errno
		err      error
	)

	if len(os.Args) < 2 {
		return errors.New("no program specified")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd = exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &unix.SysProcAttr{
		Pdeathsig: unix.SIGKILL,
		Ptrace:    true,
		Setpgid:   true,
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	pid = cmd.Process.Pid

	_, err = unix.Wait4(pid, &ws, 0, nil)
	if err != nil {
		return err
	}

	if !ws.Stopped() {
		return fmt.Errorf("expected stopped tracee, got %d", ws)
	}

	err = unix.PtraceSetOptions(
		pid,
		unix.PTRACE_O_EXITKILL|
			unix.PTRACE_O_TRACECLONE|
			unix.PTRACE_O_TRACEFORK|
			unix.PTRACE_O_TRACEVFORK,
	)
	if err != nil {
		return err
	}

	err = unix.PtraceCont(pid, 0)
	if err != nil {
		return err
	}

	for {
		pid, err = unix.Wait4(-1, &ws, 0, nil)
		if err != nil {
			return err
		}

		switch {
		case ws.Exited() && pid == cmd.Process.Pid:
			os.Exit(ws.ExitStatus())
		case ws.Signaled() && pid == cmd.Process.Pid:
			os.Exit(128 + int(ws.Signal()))
		case ws.Stopped():
			switch ws.StopSignal() {
			case unix.SIGTRAP:
				err = unix.PtraceCont(pid, 0)
				if err != nil {
					return err
				}
			case unix.SIGSYS:
				_, _, errno = unix.Syscall6(
					unix.SYS_PTRACE,
					unix.PTRACE_GETSIGINFO,
					uintptr(pid),
					0,
					uintptr(unsafe.Pointer(&sigInfo)),
					0,
					0,
				)
				if errno != 0 {
					return errno
				}

				if sigInfo.Code != 1 {
					os.Exit(128 + int(unix.SIGSYS))
				}

				archInfo, err = arch.GetInfo(runtime.GOARCH)
				if err != nil {
					return err
				}

				fmt.Fprintf(
					os.Stderr,
					"trapseccomp: blocked syscall nr=%d name=%q\n",
					sigInfo.Syscall,
					archInfo.SyscallNumbers[int(sigInfo.Syscall)],
				)

				os.Exit(128 + int(unix.SIGSYS))
			default:
				err = unix.PtraceCont(pid, int(ws.StopSignal()))
				if err != nil {
					return err
				}
			}
		}
	}
}

func main() {
	var err error

	err = run()
	if err != nil {
		fmt.Fprintf(os.Stderr, `trapseccomp: %v

Usage: trapseccomp <BINARY> [ARG]...
`, err)

		os.Exit(1)
	}
}
