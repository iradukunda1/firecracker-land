package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// For creating new vm instance
func createVmHandler(w http.ResponseWriter, r *http.Request) {

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

	opts.RootFsImage, err = opts.generateRFs(in.Name)
	if err != nil {
		log.Fatalf("failed to generate rootfs image, %s", err)
	}

	running, err := opts.createVMM(context.Background())
	if err != nil {
		log.Fatalf(err.Error())
	}

	id := uuid()
	resp := CreateResponse{
		IpAddr: opts.FcIP,
		ID:     id,
	}

	response, err := json.Marshal(&resp)
	if err != nil {
		log.Fatalf("failed to marshal json, %s", err)
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(response)

	runVms[id] = *running

	go func() {
		defer running.cancelCtx()
		// there's an error here but we ignore it for now because we terminate
		// the VM on /delete and it returns an error when it's terminated
		running.machine.Wait(running.ctx)
	}()
}

// for deleting supplied vm id
func deleteVmHandler(w http.ResponseWriter, r *http.Request) {

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

	running.machine.StopVMM()
	running.cancelCtx()

	delete(runVms, in.ID)
}
