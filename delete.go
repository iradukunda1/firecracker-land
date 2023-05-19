package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func deleteRequestHandler(w http.ResponseWriter, r *http.Request) {

	log := ctxGetLogger(r.Context())

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("failed to read body, %s", err)
	}

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
