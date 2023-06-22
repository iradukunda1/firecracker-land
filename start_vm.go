package main

import (
	"context"
	"fmt"
)

// StartVm is responsible to start vm
func StartVm(m *Firecracker) (*Firecracker, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.vm.Start(ctx); err != nil {

		m.state = StateFailed

		return m, fmt.Errorf("failed to start machine: %v", err)
	}
	defer m.vm.StopVMM()

	go func() {
		m.vm.Wait(ctx)
	}()

	m.state = StateStarted
	m.cancelCtx = cancel

	return m, nil
}
