package main

import (
	"fmt"
)

// StartVm is responsible to start vm
func StartVm(m *Firecracker) (*Firecracker, error) {

	// if m.state != StateCreated && m.vm != nil {
	if err := m.vm.Start(m.ctx); err != nil {

		m.state = StateFailed

		return m, fmt.Errorf("failed to start machine: %v", err)
	}
	defer m.vm.StopVMM()

	go func() {
		m.vm.Wait(m.ctx)
	}()

	m.state = StateStarted
	// }

	return m, nil
}
