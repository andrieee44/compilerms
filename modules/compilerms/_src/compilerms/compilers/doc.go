// Package compilers compiles and runs untrusted source code in a sandboxed
// environment.
//
// It targets hosts without Docker, KVM, systemd containers, or cgroup
// creation privileges (e.g. a shared university server), and instead
// layers several lower-privilege mechanisms to achieve isolation and
// resource limits:
//
//   - systemd-run: applies scope-level resource limits (CPU, memory, tasks)
//     without requiring cgroup creation permissions.
//   - ulimit: enforces process-level resource limits (file size, open
//     files, CPU time, etc.) as a second line of defense.
//   - bwrap (bubblewrap): provides namespace isolation and a restricted
//     filesystem jail without requiring root or container runtimes.
//   - seccomp-bpf: filters syscalls available to the sandboxed process,
//     blocking dangerous or unnecessary ones.
package compilers
