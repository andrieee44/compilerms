package compilers

import (
	"encoding/binary"
	"maps"
	"os"
	"slices"

	"github.com/elastic/go-seccomp-bpf"
	"golang.org/x/net/bpf"
)

func writeSeccompBPF(name string, whitelist map[string]struct{}) error {
	var (
		insts []bpf.Instruction
		raw   bpf.RawInstruction
		f     *os.File
		i     int
		err   error
	)

	insts, err = (&seccomp.Policy{
		DefaultAction: seccomp.ActionTrap,

		Syscalls: []seccomp.SyscallGroup{{
			Action: seccomp.ActionAllow,
			Names:  slices.Collect(maps.Keys(whitelist)),
		}},
	}).Assemble()
	if err != nil {
		return err
	}

	f, err = os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	for i = range insts {
		raw, err = insts[i].Assemble()
		if err != nil {
			return err
		}

		err = binary.Write(f, binary.NativeEndian, raw)
		if err != nil {
			return err
		}
	}

	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}
