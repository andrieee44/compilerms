local compilerms = import '../compilerms.libsonnet';

{
  allowedFiles: ['include/*.h', 'src/*.c'],

  versions: {
    as: {
      name: '/usr/bin/as',
      args: ['--version'],
    },

    gcc: {
      name: '/usr/bin/gcc',
      args: ['--version'],
    },

    ldd: {
      name: '/usr/bin/ldd',
      args: ['--version'],
    },
  },

  tests: {
    live: {
      nothing: {
        input: '',
        output: '',
        files: { 'src/main.c': 'int main(void) { return 0; }' },
      },

      hello: {
        input: '',
        output: 'Hello World!\n',

        files: {
          'src/main.c': |||
            #include <stdio.h>

            int main(void) {
              puts("Hello World!");

              return 0;
            }
          |||,
        },
      },

      header: {
        input: '4\n',
        output: 'Hello Number 4!\n',

        files: {
          'include/hello.h': |||
            #include <stdio.h>
            #include <stdlib.h>

            void hello(void);

            void hello(void) {
              int n;

              if (scanf("%d", &n) != 1) {
                exit(1);
              }

              printf("Hello Number %d!\n", n);
            }
          |||,

          'src/main.c': |||
            #include "hello.h"

            int main(void) {
              hello();

              return 0;
            }
          |||,
        },
      },
    },

    batch: self.live,
  },

  compiler: compilerms.MkProgram({
    displayName: '/usr/bin/gcc',
    shliteEnvs: ['PATH'],
    maxBytesInput: 0,
    maxBytesOutput: 8 << 10,

    systemdRunFlags: std.flattenArrays([
      ['--property', 'BindPaths={{ COMPILERMS_RUNDIR }}:/rundir'],
      ['--property', 'BindReadOnlyPaths=/lib'],
      ['--property', 'BindReadOnlyPaths=/lib64'],
      ['--property', 'BindReadOnlyPaths=/usr/bin/as'],
      ['--property', 'BindReadOnlyPaths=/usr/bin/gcc'],
      ['--property', 'BindReadOnlyPaths=/usr/bin/ld'],
      ['--property', 'BindReadOnlyPaths=/usr/include'],
      ['--property', 'BindReadOnlyPaths=/usr/lib'],
      ['--property', 'BindReadOnlyPaths=/usr/lib64'],
      ['--property', 'BindReadOnlyPaths={{ COMPILERMS_RODIR }}:/rodir'],
      ['--property', 'Environment=PATH=/usr/bin'],
      ['--property', |||
        SystemCallFilter=
          access
          arch_prctl
          brk
          chmod
          clone
          clone3
          close
          epoll_create1
          epoll_ctl
          epoll_pwait
          eventfd2
          execve
          faccessat2
          fcntl
          fstat
          futex
          getcwd
          getdents64
          getpid
          getppid
          getrandom
          getrusage
          gettid
          ioctl
          lseek
          madvise
          mmap
          mprotect
          munmap
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
          readlink
          rseq
          rt_sigaction
          rt_sigprocmask
          rt_sigreturn
          sched_getaffinity
          set_robust_list
          set_tid_address
          sigaltstack
          sysinfo
          umask
          unlink
          wait4
          waitid
          write
      |||],
    ]),

    args: std.flattenArrays([
      ['gcc'],
      ['-D_FORTIFY_SOURCE=3'],
      ['-I', '/rodir/include'],
      ['-O2'],
      ['-Wall'],
      ['-Wconversion'],
      ['-Wdouble-promotion'],
      ['-Wduplicated-branches'],
      ['-Wduplicated-cond'],
      ['-Werror'],
      ['-Wextra'],
      ['-Wformat=2'],
      ['-Wlogical-op'],
      ['-Wmissing-prototypes'],
      ['-Wnull-dereference'],
      ['-Wpedantic'],
      ['-Wshadow'],
      ['-Wsign-conversion'],
      ['-Wstrict-prototypes'],
      ['-Wundef'],
      ['-fanalyzer'],
      ['-fno-omit-frame-pointer'],
      ['-fno-sanitize-recover=all'],
      ['-fsanitize=undefined'],
      ['-fstack-protector-strong'],
      ['-g'],
      ['-lm'],
      ['-o', '/rundir/program'],
      ['/rodir/src/*'],
    ]),
  }),

  program: {
    local cfgProgram = {
      displayName: '/rundir/program',
      shliteEnvs: [],
      maxBytesInput: 8 << 10,
      maxBytesOutput: 8 << 10,
      args: ['/rundir/program'],

      systemdRunFlags: std.flattenArrays([
        ['--property', 'BindReadOnlyPaths=/lib'],
        ['--property', 'BindReadOnlyPaths=/lib64'],
        ['--property', 'BindReadOnlyPaths=/usr/lib'],
        ['--property', 'BindReadOnlyPaths=/usr/lib64'],
        ['--property', 'BindReadOnlyPaths={{ COMPILERMS_RUNDIR }}:/rundir'],
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
            getpid
            getppid
            getrandom
            gettid
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
            set_robust_list
            set_tid_address
            sigaltstack
            waitid
            write
        |||],
      ]),
    },

    live: compilerms.MkProgram(cfgProgram {
      systemdRunFlags+: ['--property', 'RuntimeMaxSec=180'],
    }),

    batch: compilerms.MkProgram(cfgProgram),
  },
}
