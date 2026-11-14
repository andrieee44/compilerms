local systemdRun = {
  name: '/usr/bin/systemd-run',

  args: std.flattenArrays([
    ['--collect'],
    ['--nice', '19'],
    ['--no-ask-password'],
    ['--pipe'],
    ['--property', 'CPUQuota=100%'],
    ['--property', 'CapabilityBoundingSet='],
    ['--property', 'IgnoreSIGPIPE=false'],
    ['--property', 'KeyringMode=private'],
    ['--property', 'LimitCORE=0'],
    ['--property', 'LimitCPU=5'],
    ['--property', 'LimitFSIZE=25M'],
    ['--property', 'LimitLOCKS=0'],
    ['--property', 'LimitMEMLOCK=0'],
    ['--property', 'LimitMSGQUEUE=0'],
    ['--property', 'LimitNICE=19'],
    ['--property', 'LimitRTPRIO=0'],
    ['--property', 'LimitRTTIME=0'],
    ['--property', 'LimitSIGPENDING=64'],
    ['--property', 'LockPersonality=true'],
    ['--property', 'MemoryHigh=950M'],
    ['--property', 'MemoryMax=1024M'],
    ['--property', 'MountAPIVFS=false'],
    ['--property', 'NoNewPrivileges=true'],
    ['--property', 'PrivateDevices=true'],
    ['--property', 'PrivateIPC=true'],
    ['--property', 'PrivateMounts=true'],
    ['--property', 'PrivateNetwork=true'],
    ['--property', 'PrivatePIDs=true'],
    ['--property', 'PrivateTmp=disconnected'],
    ['--property', 'PrivateUsers=true'],
    ['--property', 'ProcSubset=pid'],
    ['--property', 'ProtectClock=true'],
    ['--property', 'ProtectControlGroups=strict'],
    ['--property', 'ProtectHostname=true:compilerms'],
    ['--property', 'ProtectKernelLogs=true'],
    ['--property', 'ProtectKernelModules=true'],
    ['--property', 'ProtectKernelTunables=true'],
    ['--property', 'ProtectProc=noaccess'],
    ['--property', 'RemoveIPC=true'],
    ['--property', 'RestrictNamespaces=true'],
    ['--property', 'RestrictRealtime=true'],
    ['--property', 'RestrictSUIDSGID=true'],
    ['--property', 'RuntimeMaxSec=5'],
    ['--property', 'SystemCallArchitectures=native'],
    ['--property', 'SystemCallErrorNumber=EPERM'],
    ['--property', 'SystemCallFilter=access'],
    ['--property', 'TasksMax=64'],
    ['--property', 'TemporaryFileSystem=/:ro'],
    ['--property', 'UMask=0077'],
    ['--property', 'WorkingDirectory=/'],
    ['--quiet'],
    ['--user'],
  ]),
};

local strace = {
  name: '/usr/bin/strace',

  systemdRunFlags: std.flattenArrays([
    ['--property', 'BindReadOnlyPaths=/lib'],
    ['--property', 'BindReadOnlyPaths=/lib64'],
    ['--property', 'BindReadOnlyPaths=/usr/lib'],
    ['--property', 'BindReadOnlyPaths=/usr/lib64'],
    ['--property', 'BindReadOnlyPaths=' + self.name],
    ['--property', 'SystemCallFilter='],
  ]),

  args: std.flattenArrays([
    ['--follow-forks'],
    ['--summary-columns', 'name'],
    ['--summary-only'],
    ['--summary-sort-by', 'name'],
  ]),
};

local shlite = {
  name: '/home/andrieee44/.local/bin/shlite',
  systemdRunFlags: ['--property', 'BindReadOnlyPaths=' + self.name],
};

{
  MkProgram(cfg):: {
    assert std.isObject(cfg),
    assert std.isString(cfg.displayName),
    assert std.isArray(cfg.shliteEnvs),
    assert std.isNumber(cfg.maxBytesInput),
    assert std.isNumber(cfg.maxBytesOutput),
    assert std.isArray(cfg.systemdRunFlags),
    assert std.isArray(cfg.args),

    local mkArgs(debug) = std.flattenArrays([
      systemdRun.args,
      shlite.systemdRunFlags,
      cfg.systemdRunFlags,
      if debug then strace.systemdRunFlags else [],
      ['--'],
      if debug then [strace.name] + strace.args + ['--'] else [],
      [shlite.name, std.join(':', cfg.shliteEnvs)],
      cfg.args,
    ]),

    displayName: cfg.displayName,
    name: systemdRun.name,
    maxBytesInput: cfg.maxBytesInput,
    maxBytesOutput: cfg.maxBytesOutput,

    args: {
      prod: mkArgs(false),
      debug: mkArgs(true),
    },
  },
}
