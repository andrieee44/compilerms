local compilerms = import '../compilerms.libsonnet';

{
  allowedFiles: ['main.sh'],

  versions: {
    bash: {
      name: '/usr/bin/bash',
      args: ['--version'],
    },
  },

  tests: {
    live: {
      nothing: {
        input: 'exit\n',
        output: '',
        files: {},
      },

      hello: {
        input: 'echo "Hello World!"\nexit\n',
        output: 'Hello World!\n',
        files: {},
      },

      main: {
        input: 'source /rodir/main.sh\nexit\n',
        output: 'Hello World!\n',
        files: { 'main.sh': 'echo "Hello World!"' },
      },
    },

    batch: {
      nothing: {
        input: '',
        output: '',
        files: { 'main.sh': '' },
      },

      hello: {
        input: '',
        output: 'Hello World!\n',
        files: { 'main.sh': 'echo "Hello World!"' },
      },
    },
  },

  compiler: null,

  program: {
    local cfgProgram = {
      displayName: '/usr/bin/bash',
      shliteEnvs: ['PATH'],
      maxBytesInput: 8 << 10,
      maxBytesOutput: 16 << 10,
      args: ['/usr/bin/bash', '--noprofile', '--norc', '/rodir/main.sh'],

      systemdRunFlags: std.flattenArrays([
        ['--property', 'BindPaths={{ COMPILERMS_RUNDIR }}:/rundir'],
        ['--property', 'BindReadOnlyPaths=/lib'],
        ['--property', 'BindReadOnlyPaths=/lib64'],
        ['--property', 'BindReadOnlyPaths=/usr/bin/bash'],
        ['--property', 'BindReadOnlyPaths=/usr/lib'],
        ['--property', 'BindReadOnlyPaths=/usr/lib64'],
        ['--property', 'BindReadOnlyPaths={{ COMPILERMS_RODIR }}:/rodir'],
        ['--property', 'Environment=PATH=/usr/bin'],
        ['--property', 'WorkingDirectory=/rundir'],
        ['--property', |||
          SystemCallFilter=
            access
            arch_prctl
            brk
            clone
            close
            execve
            fcntl
            fstat
            futex
            getcwd
            getpgrp
            getpid
            getppid
            getrandom
            getresgid
            getresuid
            gettid
            ioctl
            lseek
            madvise
            mmap
            mprotect
            nanosleep
            newfstatat
            openat
            pidfd_open
            pidfd_send_signal
            pipe2
            prctl
            pread64
            prlimit64
            read
            rseq
            rt_sigaction
            rt_sigprocmask
            rt_sigreturn
            sched_getaffinity
            sched_yield
            set_robust_list
            set_tid_address
            sigaltstack
            tgkill
            uname
            waitid
        |||],
      ]),
    },

    batch: compilerms.MkProgram(cfgProgram),

    live: compilerms.MkProgram(cfgProgram {
      args: cfgProgram.args[:-1],
      systemdRunFlags+: ['--property', 'RuntimeMaxSec=180'],
    }),
  },
}
