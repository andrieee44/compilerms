package compilers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

type GCCOpts struct {
	Headers map[string]string `json:"headers"`
	Sources map[string]string `json:"source"`
	Stdin   string            `json:"stdin"`
}

var (
	gccCompilerWhitelist map[string]struct{} = map[string]struct{}{
		"access":          {},
		"arch_prctl":      {},
		"brk":             {},
		"chmod":           {},
		"clone3":          {},
		"close":           {},
		"execve":          {},
		"exit_group":      {},
		"faccessat2":      {},
		"fcntl":           {},
		"fstat":           {},
		"getcwd":          {},
		"getrandom":       {},
		"getrusage":       {},
		"ioctl":           {},
		"lseek":           {},
		"mmap":            {},
		"mprotect":        {},
		"munmap":          {},
		"newfstatat":      {},
		"openat":          {},
		"pipe2":           {},
		"pread64":         {},
		"prlimit64":       {},
		"read":            {},
		"readlink":        {},
		"rseq":            {},
		"rt_sigaction":    {},
		"rt_sigprocmask":  {},
		"set_robust_list": {},
		"set_tid_address": {},
		"sysinfo":         {},
		"umask":           {},
		"unlink":          {},
		"vfork":           {},
		"wait4":           {},
		"write":           {},
	}

	gccProgramWhitelist map[string]struct{} = map[string]struct{}{
		"access":          {},
		"arch_prctl":      {},
		"brk":             {},
		"clock_gettime":   {},
		"close":           {},
		"execve":          {},
		"exit_group":      {},
		"fstat":           {},
		"futex":           {},
		"getpid":          {},
		"getrandom":       {},
		"gettid":          {},
		"lseek":           {},
		"madvise":         {},
		"mmap":            {},
		"mprotect":        {},
		"munmap":          {},
		"newfstatat":      {},
		"openat":          {},
		"pread64":         {},
		"prlimit64":       {},
		"read":            {},
		"rseq":            {},
		"rt_sigaction":    {},
		"rt_sigprocmask":  {},
		"rt_sigreturn":    {},
		"set_robust_list": {},
		"set_tid_address": {},
		"sigaltstack":     {},
		"statx":           {},
		"tgkill":          {},
		"wait4":           {},
		"write":           {},
		"writev":          {},
	}
)

func GCC(ctx context.Context, opts GCCOpts) (Output, error) {
	var (
		tmpdir, source, compilerBPF, programBPF string
		deferFn                                 func()
		sources                                 []string
		output                                  Output
		err                                     error
	)

	tmpdir, deferFn, err = mkdirTemp("compilerms-gcc-*")
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	defer deferFn()

	err = mkdirFiles(filepath.Join(tmpdir, "include"), opts.Headers)
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	err = mkdirFiles(filepath.Join(tmpdir, "source"), opts.Sources)
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	sources = make([]string, 0, len(opts.Sources))
	for source = range opts.Sources {
		sources = append(sources, filepath.Join("/", "source", source))
	}

	compilerBPF = filepath.Join(tmpdir, "compiler.bpf")
	err = writeSeccompBPF(compilerBPF, gccCompilerWhitelist)
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	programBPF = filepath.Join(tmpdir, "program.bpf")
	err = writeSeccompBPF(programBPF, gccProgramWhitelist)
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	output, err = run(
		ctx,
		nil,
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
					--new-session \
					--unshare-all \
					--chdir / \
					--dev /dev \
					--hostname compilerms \
					--tmpfs /tmp \
					--bind %s / \
					--ro-bind /bin /bin \
					--ro-bind /lib /lib \
					--ro-bind /lib64 /lib64 \
					--ro-bind /usr /usr \
					--setenv PATH /usr/bin:/bin \
					--setenv TMPDIR /tmp \
					--seccomp 3 3< %s \
					-- gcc \
						-Wall \
						-Werror \
						-Wextra \
						-Wpedantic \
						-I /include \
						-o /program \
						%s
		`, shellescape.Quote(tmpdir),
			shellescape.Quote(compilerBPF),
			shellescape.QuoteCommand(sources)),
	)
	if err != nil {
		return output, fmt.Errorf("%w: %w", ErrCompiler, err)
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
		"-p", "RuntimeMaxSec=3",
		"-p", "TasksMax=32",
		"--", "sh", "-c",
		fmt.Sprintf(`
			set -eu

			ulimit -c 0
			ulimit -d 524288
			ulimit -f 512
			ulimit -l 0
			ulimit -m 1048576
			ulimit -n 32
			ulimit -r 0
			ulimit -s 8192
			ulimit -t 3
			ulimit -v 1048576

			exec trapseccomp \
				bwrap \
					--clearenv \
					--die-with-parent \
					--new-session \
					--unshare-all \
					--chdir / \
					--dev /dev \
					--hostname compilerms \
					--tmpfs /tmp \
					--bind %s / \
					--ro-bind /lib /lib \
					--ro-bind /lib64 /lib64 \
					--ro-bind /usr/lib /usr/lib \
					--ro-bind /usr/lib64 /usr/lib64 \
					--setenv TMPDIR /tmp \
					--seccomp 3 3< %s \
					-- /program
		`, shellescape.Quote(tmpdir), shellescape.Quote(programBPF)),
	)
	if err != nil {
		return output, fmt.Errorf("%w: %w", ErrProgram, err)
	}

	return output, nil
}
