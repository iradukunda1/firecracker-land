package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi"
)

var runVms map[string]*Firecracker = make(map[string]*Firecracker)
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

	id := uuid()

	opts := getOptions(ipByte, *in)

	opts.RootFsImage, err = opts.GenerateRFs(in.Name)
	if err != nil {
		fmt.Printf("failed to generate rootfs image, %s", err)
		return
	}

	m, err := opts.createVMM(r.Context(), id)
	if err != nil {
		fmt.Printf("failed to start and create vm %v", err)
		return
	}

	resp := CreateResponse{
		Name:   in.Name,
		State:  m.state,
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

	m, err = StartVm(m)
	if err != nil {
		fmt.Printf("failed to start vm, %s", err)
		return
	}

	runVms[id] = m

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

	if err := running.vm.Shutdown(running.ctx); err != nil {
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

	if err := running.vm.PauseVM(running.ctx); err != nil {
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

	if err := running.vm.ResumeVM(running.ctx); err != nil {
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
		pid, _ := v.vm.PID()
		resp = append(resp, CreateResponse{
			Name:   v.Name,
			State:  v.state,
			IpAddr: string(v.vm.Cfg.MmdsAddress),
			ID:     v.vm.Cfg.VMID,
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

// For getting vm details using supplied vm id
func InfoVmHandler(w http.ResponseWriter, r *http.Request) {

	log := ctxGetLogger(r.Context())

	id := chi.URLParam(r, "vm_id")

	running, ok := runVms[id]
	if !ok {
		res := &responseMessage{
			Message: fmt.Sprintf("the vm machine with this id %s is not exist", id),
		}
		resp, _ := json.Marshal(&res)
		w.Header().Add("Content-Type", "application/json")
		w.Write(resp)
		return
	}

	resp := CreateResponse{
		Name:   running.Name,
		State:  running.state,
		IpAddr: string(running.vm.Cfg.MmdsAddress),
		ID:     running.vm.Cfg.VMID,
		Agent:  running.Agent,
	}

	response, err := json.Marshal(&resp)
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(response)
}
