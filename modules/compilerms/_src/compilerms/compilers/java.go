package compilers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

type JavaOpts struct {
	Sources map[string]string `json:"sources" validate:"gt=0,dive,keys,excludesrune=/,endkeys"`
	Stdin   string            `json:"stdin"`
}

var (
	javaCompilerWhitelist map[string]struct{} = map[string]struct{}{
		"access":            {},
		"arch_prctl":        {},
		"brk":               {},
		"clock_getres":      {},
		"clock_gettime":     {},
		"clock_nanosleep":   {},
		"clone3":            {},
		"close":             {},
		"connect":           {},
		"dup":               {},
		"execve":            {},
		"exit":              {},
		"exit_group":        {},
		"fcntl":             {},
		"fstat":             {},
		"futex":             {},
		"getcwd":            {},
		"getdents64":        {},
		"geteuid":           {},
		"getpid":            {},
		"getrandom":         {},
		"gettid":            {},
		"getuid":            {},
		"ioctl":             {},
		"lseek":             {},
		"madvise":           {},
		"mkdir":             {},
		"mmap":              {},
		"mprotect":          {},
		"munmap":            {},
		"newfstatat":        {},
		"openat":            {},
		"prctl":             {},
		"pread64":           {},
		"prlimit64":         {},
		"read":              {},
		"readlink":          {},
		"readlinkat":        {},
		"restart_syscall":   {},
		"rseq":              {},
		"rt_sigaction":      {},
		"rt_sigprocmask":    {},
		"rt_sigreturn":      {},
		"sched_getaffinity": {},
		"sched_yield":       {},
		"set_robust_list":   {},
		"set_tid_address":   {},
		"setsockopt":        {},
		"socket":            {},
		"stat":              {},
		"statfs":            {},
		"statx":             {},
		"sysinfo":           {},
		"uname":             {},
		"wait4":             {},
		"write":             {},
		"writev":            {},
	}

	javaProgramWhitelist map[string]struct{} = map[string]struct{}{
		"access":            {},
		"arch_prctl":        {},
		"brk":               {},
		"clock_getres":      {},
		"clock_nanosleep":   {},
		"clone3":            {},
		"close":             {},
		"connect":           {},
		"execve":            {},
		"exit":              {},
		"exit_group":        {},
		"fcntl":             {},
		"fstat":             {},
		"futex":             {},
		"getcwd":            {},
		"getpid":            {},
		"getrandom":         {},
		"gettid":            {},
		"getuid":            {},
		"ioctl":             {},
		"lseek":             {},
		"madvise":           {},
		"mmap":              {},
		"mprotect":          {},
		"munmap":            {},
		"newfstatat":        {},
		"openat":            {},
		"prctl":             {},
		"pread64":           {},
		"prlimit64":         {},
		"read":              {},
		"readlink":          {},
		"readlinkat":        {},
		"restart_syscall":   {},
		"rseq":              {},
		"rt_sigaction":      {},
		"rt_sigprocmask":    {},
		"rt_sigreturn":      {},
		"sched_getaffinity": {},
		"sched_yield":       {},
		"set_robust_list":   {},
		"set_tid_address":   {},
		"socket":            {},
		"stat":              {},
		"statfs":            {},
		"statx":             {},
		"sysinfo":           {},
		"uname":             {},
		"wait4":             {},
		"write":             {},
	}
)

func Java(ctx context.Context, opts JavaOpts) (Output, error) {
	var (
		tmpdir, source, compilerBPF, programBPF string
		sources                                 []string
		output                                  Output
		err                                     error
	)

	err = validate.Struct(opts)
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}

	tmpdir, err = os.MkdirTemp(os.TempDir(), "compilerms-java-*")
	if err != nil {
		return Output{}, err
	}

	defer os.RemoveAll(tmpdir) //nolint:errcheck

	err = mkdirFiles(filepath.Join(tmpdir, "sources"), opts.Sources)
	if err != nil {
		return Output{}, err
	}

	sources = make([]string, 0, len(opts.Sources))
	for source = range opts.Sources {
		sources = append(sources, filepath.Join("/", "sources", source))
	}

	compilerBPF = filepath.Join(tmpdir, "compiler.bpf")
	err = writeSeccompBPF(compilerBPF, javaCompilerWhitelist)
	if err != nil {
		return Output{}, err
	}

	programBPF = filepath.Join(tmpdir, "program.bpf")
	err = writeSeccompBPF(programBPF, javaProgramWhitelist)
	if err != nil {
		return Output{}, err
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

			home="$(dirname "$(dirname "$(readlink -e "$(which java)")")")"

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
					--proc /proc \
					--tmpfs /tmp \
					--ro-bind "$home/bin/javac" "$home/bin/javac" \
					--ro-bind "$home/lib" "$home/lib" \
					--ro-bind /lib /lib \
					--ro-bind /lib64 /lib64 \
					--ro-bind /usr/lib /usr/lib \
					--ro-bind /usr/lib64 /usr/lib64 \
					--chdir / \
					--setenv PATH "$home/bin" \
					--setenv TMPDIR /tmp \
					--seccomp 3 3< %s \
					-- javac \
						-J-XX:+UseSerialGC \
						-J-XX:-UsePerfData \
						-J-XX:ActiveProcessorCount=1 \
						-J-XX:CICompilerCount=1 \
						-J-XX:CompressedClassSpaceSize=32m \
						-J-XX:ConcGCThreads=1 \
						-J-XX:InitialCodeCacheSize=2m \
						-J-XX:MaxDirectMemorySize=16m \
						-J-XX:MaxMetaspaceSize=64m \
						-J-XX:ParallelGCThreads=1 \
						-J-XX:ReservedCodeCacheSize=16m \
						-J-XX:ThreadStackSize=256 \
						-J-XX:TieredStopAtLevel=1 \
						-J-Xms8m \
						-J-Xmx384m \
						-J-Xshare:off \
						-J-Xss256k \
						-Werror \
						-Xlint \
						-d /out \
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

			home="$(dirname "$(dirname "$(readlink -e "$(which java)")")")"

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
					--proc /proc \
					--tmpfs /tmp \
					--ro-bind "$home/bin/java" "$home/bin/java" \
					--ro-bind "$home/lib" "$home/lib" \
					--ro-bind /lib /lib \
					--ro-bind /lib64 /lib64 \
					--ro-bind /usr/lib /usr/lib \
					--ro-bind /usr/lib64 /usr/lib64 \
					--chdir / \
					--setenv PATH "$home/bin" \
					--setenv TMPDIR /tmp \
					--seccomp 3 3< %s \
					-- java \
						-XX:+UseSerialGC \
						-XX:-UsePerfData \
						-XX:ActiveProcessorCount=1 \
						-XX:CICompilerCount=1 \
						-XX:CompressedClassSpaceSize=32m \
						-XX:ConcGCThreads=1 \
						-XX:InitialCodeCacheSize=2m \
						-XX:MaxDirectMemorySize=16m \
						-XX:MaxMetaspaceSize=64m \
						-XX:ParallelGCThreads=1 \
						-XX:ReservedCodeCacheSize=16m \
						-XX:ThreadStackSize=256 \
						-XX:TieredStopAtLevel=1 \
						-Xms8m \
						-Xmx384m \
						-Xshare:off \
						-Xss256k \
						-cp /out \
						Main
		`, shellescape.Quote(tmpdir), shellescape.Quote(programBPF)),
	)
	if err != nil {
		return output, fmt.Errorf("%w: %w", ErrProgram, err)
	}

	return output, nil
}
