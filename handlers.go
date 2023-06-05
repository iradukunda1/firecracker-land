package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

var runVms map[string]Firecracker = make(map[string]Firecracker)
var ipByte byte = 3

// For creating new vm instance
func CreateVmHandler(w http.ResponseWriter, r *http.Request) {

	log := ctxGetLogger(r.Context())

	ipByte += 1
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("failed to read body, %s", err)
	}
	defer r.Body.Close()

	in := new(CreateRequest)

	if err := json.Unmarshal([]byte(body), in); err != nil {
		log.Fatalf("error during reading passed request body: %v", err.Error())
	}

	opts := getOptions(ipByte, *in)

	// opts.RootFsImage = in.DockerImage

	opts.RootFsImage, err = opts.GenerateRFs(in.Name)
	if err != nil {
		log.Fatalf("failed to generate rootfs image, %s", err)
	}

	id := uuid()
	m, err := opts.createVMM(context.Background(), id)
	if err != nil {
		log.Fatalf(err.Error())
	}

	resp := CreateResponse{
		Name:   in.Name,
		IpAddr: opts.FcIP,
		ID:     id,
		Agent:  m.Agent,
	}

	response, err := json.Marshal(&resp)
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(response)

	runVms[id] = *m

	// g := errgroup.Group{}

	// g.Go(func() error {

	// 	if err := m.machine.Start(m.ctx); err != nil {
	// 		log.Fatalf("failed to start machine: %v", err)
	// 	}
	// 	defer m.cancelCtx()

	// 	// there's an error here but we ignore it for now because we terminate
	// 	// the VM on /delete and it returns an error when it's terminated
	// 	if err := m.machine.Wait(m.ctx); err != nil {
	// 		return err
	// 	}

	// 	return nil
	// })

	// if err := g.Wait(); err != nil {
	// 	log.Fatalf("failed to start machine: %v", err)
	// }

}

// for deleting supplied vm id
func DeleteVmHandler(w http.ResponseWriter, r *http.Request) {

	log := ctxGetLogger(r.Context())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("failed to read body, %s", err)
	}
	defer r.Body.Close()

	in := new(DeleteRequest)

	err = json.Unmarshal([]byte(body), in)
	if err != nil {
		log.Fatalf("error during reading passed request body: %v", err.Error())
	}

	running := runVms[in.ID]

	if err := running.machine.Shutdown(running.ctx); err != nil {
		log.Fatalf("failed to delete vm, %s", err)
	}

	res, err := json.Marshal(&responseMessage{Message: "vm deleted successfully"})
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(res)

	delete(runVms, in.ID)
}

// For stopping vm using supplied vm id
func StopVmHandler(w http.ResponseWriter, r *http.Request) {

	log := ctxGetLogger(r.Context())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("failed to read body, %s", err)
	}
	defer r.Body.Close()

	in := new(DeleteRequest)

	json.Unmarshal([]byte(body), in)
	if err != nil {
		log.Fatalf("error during reading passed request body: %v", err.Error())
	}

	running := runVms[in.ID]

	if err := running.machine.PauseVM(running.ctx); err != nil {
		log.Fatalf("failed to pause vm, %s", err)
	}
	defer running.cancelCtx()

	res, err := json.Marshal(&responseMessage{Message: "vm stopped successfully"})
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(res)
}

// For resuming vm using supplied vm id
func ResumeVmHandler(w http.ResponseWriter, r *http.Request) {

	log := ctxGetLogger(r.Context())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("failed to read body, %s", err)
	}
	defer r.Body.Close()

	in := new(DeleteRequest)

	json.Unmarshal([]byte(body), in)
	if err != nil {
		log.Fatalf("error during reading passed request body: %v", err.Error())
	}

	running := runVms[in.ID]

	if err := running.machine.ResumeVM(running.ctx); err != nil {
		log.Fatalf("failed to resume vm, %s", err)
	}

	res, err := json.Marshal(&responseMessage{Message: "vm resumed successfully"})
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(res)
}

// For getting all running vms
func ListVmsHandler(w http.ResponseWriter, r *http.Request) {

	log := ctxGetLogger(r.Context())

	var resp []CreateResponse = make([]CreateResponse, 0)

	for _, v := range runVms {
		pid, _ := v.machine.PID()
		resp = append(resp, CreateResponse{
			IpAddr: string(v.machine.Cfg.MmdsAddress),
			ID:     v.machine.Cfg.VMID,
			PID:    int64(pid),
		})
	}

	response, err := json.Marshal(&resp)
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(response)
}
