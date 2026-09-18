package compilers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

type SQLite3tmpOpts struct {
	Stdin string `json:"stdin" validate:"required"`
}

var sqlite3tmpProgramWhitelist map[string]struct{} = map[string]struct{}{
	"arch_prctl":        {},
	"brk":               {},
	"clone":             {},
	"close":             {},
	"epoll_create1":     {},
	"epoll_ctl":         {},
	"epoll_pwait":       {},
	"eventfd2":          {},
	"execve":            {},
	"exit_group":        {},
	"fcntl":             {},
	"futex":             {},
	"getpid":            {},
	"gettid":            {},
	"madvise":           {},
	"mmap":              {},
	"mprotect":          {},
	"munmap":            {},
	"nanosleep":         {},
	"open":              {},
	"openat":            {},
	"prctl":             {},
	"prlimit64":         {},
	"read":              {},
	"readlinkat":        {},
	"restart_syscall":   {},
	"rt_sigaction":      {},
	"rt_sigprocmask":    {},
	"rt_sigreturn":      {},
	"sched_getaffinity": {},
	"sched_yield":       {},
	"set_tid_address":   {},
	"sigaltstack":       {},
	"tgkill":            {},
	"wait4":             {},
	"write":             {},
}

func SQLite3tmp(ctx context.Context, opts SQLite3tmpOpts) (Output, error) {
	var (
		tmpdir, programBPF string
		output             Output
		err                error
	)

	err = validate.Struct(opts)
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}

	tmpdir, err = os.MkdirTemp(os.TempDir(), "compilerms-sqlite3tmp-*")
	if err != nil {
		return Output{}, err
	}

	defer os.RemoveAll(tmpdir) //nolint:errcheck

	programBPF = filepath.Join(tmpdir, "program.bpf")
	err = writeSeccompBPF(programBPF, sqlite3tmpProgramWhitelist)
	if err != nil {
		return Output{}, err
	}

	output, err = run(
		ctx,
		strings.NewReader(opts.Stdin),
		"systemd-run",
		"--collect",
		"--quiet",
		"--scope",
		"--user",
		"-p", "CPUQuota=100%",
		"-p", "MemoryHigh=900M",
		"-p", "MemoryMax=1024M",
		"-p", "RuntimeMaxSec=5",
		"-p", "TasksMax=32",
		"--", "sh", "-c",
		fmt.Sprintf(`
			set -eu

			ulimit -c 0
			ulimit -d 524288
			ulimit -f 2048
			ulimit -l 0
			ulimit -m 1048576
			ulimit -n 64
			ulimit -r 0
			ulimit -s 8192
			ulimit -t 5
			ulimit -v 1048576

			exec trapseccomp \
				bwrap \
					--clearenv \
					--die-with-parent \
					--disable-userns \
					--new-session \
					--unshare-all \
					--unshare-user \
					--hostname compilerms \
					--bind %s / \
					--dev /dev \
					--tmpfs /tmp \
					--ro-bind "$(which sqlite3tmp)" /sqlite3tmp \
					--chdir / \
					--setenv TMPDIR /tmp \
					--seccomp 3 3< %s \
					-- /sqlite3tmp
		`, shellescape.Quote(tmpdir), shellescape.Quote(programBPF)),
	)
	if err != nil {
		return output, fmt.Errorf("%w: %w", ErrProgram, err)
	}

	return output, nil
}
