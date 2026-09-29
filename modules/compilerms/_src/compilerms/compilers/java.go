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
		roDir, runDir, source string
		sources               []string
		output                Output
		err                   error
	)

	err = validate.Struct(opts)
	if err != nil {
		return Output{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}

	roDir, err = os.MkdirTemp(os.TempDir(), "compilerms-java-readonly-*")
	if err != nil {
		return Output{}, err
	}

	defer os.RemoveAll(roDir) //nolint:errcheck

	err = mkdirWithFiles(filepath.Join(roDir, "sources"), opts.Sources)
	if err != nil {
		return Output{}, err
	}

	sources = make([]string, 0, len(opts.Sources))
	for source = range opts.Sources {
		sources = append(sources, filepath.Join("/", "sources", source))
	}

	err = writeSeccompBPF(
		filepath.Join(roDir, "seccomp", "compiler.bpf"),
		javaCompilerWhitelist,
	)
	if err != nil {
		return Output{}, err
	}

	err = writeSeccompBPF(
		filepath.Join(roDir, "seccomp", "program.bpf"),
		javaProgramWhitelist,
	)
	if err != nil {
		return Output{}, err
	}

	runDir, err = os.MkdirTemp(os.TempDir(), "compilerms-java-runtime-*")
	if err != nil {
		return Output{}, err
	}

	defer os.RemoveAll(runDir) //nolint:errcheck

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

			JAVA_HOME="$(dirname "$(dirname "$(readlink -e "$(which java)")")")"
			RODIR=%s
			RUNDIR=%s

			exec trapseccomp \
				bwrap \
					--clearenv \
					--die-with-parent \
					--disable-userns \
					--new-session \
					--unshare-all \
					--unshare-user \
					--hostname compilerms \
					--bind "$RUNDIR" / \
					--dev /dev \
					--proc /proc \
					--tmpfs /tmp \
					--ro-bind "$JAVA_HOME/bin/javac" "$JAVA_HOME/bin/javac" \
					--ro-bind "$JAVA_HOME/lib" "$JAVA_HOME/lib" \
					--ro-bind "$RODIR/sources" /sources \
					--ro-bind /lib /lib \
					--ro-bind /lib64 /lib64 \
					--ro-bind /usr/lib /usr/lib \
					--ro-bind /usr/lib64 /usr/lib64 \
					--chdir / \
					--setenv PATH "$JAVA_HOME/bin" \
					--setenv TMPDIR /tmp \
					--seccomp 3 3< "$RODIR/seccomp/compiler.bpf" \
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
		`, shellescape.Quote(roDir),
			shellescape.Quote(runDir),
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

			JAVA_HOME="$(dirname "$(dirname "$(readlink -e "$(which java)")")")"
			RODIR=%s
			RUNDIR=%s

			exec trapseccomp \
				bwrap \
					--clearenv \
					--die-with-parent \
					--disable-userns \
					--new-session \
					--unshare-all \
					--unshare-user \
					--hostname compilerms \
					--bind "$RUNDIR" / \
					--dev /dev \
					--proc /proc \
					--tmpfs /tmp \
					--ro-bind "$JAVA_HOME/bin/java" "$JAVA_HOME/bin/java" \
					--ro-bind "$JAVA_HOME/lib" "$JAVA_HOME/lib" \
					--ro-bind /lib /lib \
					--ro-bind /lib64 /lib64 \
					--ro-bind /usr/lib /usr/lib \
					--ro-bind /usr/lib64 /usr/lib64 \
					--chdir / \
					--setenv PATH "$JAVA_HOME/bin" \
					--setenv TMPDIR /tmp \
					--seccomp 3 3< "$RODIR/seccomp/program.bpf" \
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
		`, shellescape.Quote(roDir), shellescape.Quote(runDir)),
	)
	if err != nil {
		return output, fmt.Errorf("%w: %w", ErrProgram, err)
	}

	return output, nil
}
