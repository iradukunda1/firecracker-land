# Firecracker-land

This repository contains a GoLang project that leverages Firecracker to create and manage virtual machines (VMs). Follow the instructions below to set up and run the project. (This is prototype of firecracker demo testing golang codes)

## Prerequisites

Before running this project, ensure that you have the following dependencies installed:

* Go: [Installation Guide](https://golang.org/doc/install)
* Task: [Installation Guide](https://taskfile.dev/#/installation)
* Firecracker & Jailer binary: [Installation Guide](https://github.com/firecracker-microvm/firecracker#getting-started)

## Getting Started

1. Clone the repository:

   ```
   git clone https://github.com/iradukunda1/firecracker-land.git
   cd firecracker-land
   ```
2. To pull a customized kernel image into the current location of the project, use the `kernel` task:

   ```
   task pull-kernel
   ```
3. Build the binary using the `build` task:

```
   task build
```

   this will generate an executable binary file in the project's root directory.

4. Run below command to generate init process and initrd

```
task gen-init && task gen-initrd
```

5. Run the project using the `run` task:

```
   task run
```

The application will start running on `localhost:8080`.

## Available Endpoints

The following endpoints are available for interacting with the application:

* `/api/create`: This endpoint is used to create a new VM. It expects the location or name of you docker container image as input and name.
* `/api/delete`: This endpoint is used to delete a VM. It requires the VM ID to be provided as the request body.

1. Create a VM using `/api/create`:

   ```

   curl -X POST -H "Content-Type: application/json" -d '{"docker-image": "Docker container name","name":"my-hello-vm"}' http://localhost:8080/api/create

   ```

   Replace `/path/to/rootfs.img` with the actual path to the rootfs image you want to use.
2. Delete a VM using `/api/delete`:

   ```

   curl -X POST -H "Content-Type: application/json" -d '{"id": "uuid-generated"}' http://localhost:8080/api/delete

   ```

Replace `uuid-generated` with the ID of the VM you want to delete.

Please note that you need to have the server running (task run) before executing these curl commands. Make sure to replace localhost:8080 with the appropriate host and port if you are running the server on a different location.

Feel free to modify the request bodies or endpoints as needed for your testing purposes.
