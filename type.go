package main

import "sync"

// MachineCfg provides machine configuration options.
type MachineCfg struct {
	CNINetworkName    string `json:"CniNetworkName" mapstructure:"CniNetworkName"`
	CPU               int64  `json:"CPU" mapstructure:"CPU"`
	CPUTemplate       string `json:"CPUTemplate" mapstructure:"CPUTemplate"`
	HTEnabled         bool   `json:"HTEnabled" mapstructure:"HTEnabled"`
	IPAddress         string `json:"IPAddress" mapstructure:"IPAddress"`
	KernelArgs        string `json:"KernelArgs" mapstructure:"KernelArgs"`
	Mem               int64  `json:"Mem" mapstructure:"Mem"`
	NoMMDS            bool   `json:"NoMMDS" mapstructure:"NoMMDS"` // TODO: remove
	RootDrivePartUUID string `json:"RootDrivePartuuid" mapstructure:"RootDrivePartuuid"`
	SSHUser           string `json:"SSHUser" mapstructure:"SSHUser"`
	VMLinuxID         string `json:"VMLinux" mapstructure:"VMLinux"`

	LogFcHTTPCalls                 bool `json:"LogFirecrackerHTTPCalls" mapstructure:"LogFirecrackerHTTPCalls"`
	ShutdownGracefulTimeoutSeconds int  `json:"ShutdownGracefulTimeoutSeconds" mapstructure:"ShutdownGracefulTimeoutSeconds"`

	daemonize      bool
	kernelOverride string
	rootfsOverride string
}

// JailingFirecrackerConfig represents Jailerspecific configuration options.
type JailingFirecrackerConfig struct {
	sync.Mutex

	BinaryFirecracker string `json:"BinaryFirecracker" mapstructure:"BinaryFirecracker"`
	BinaryJailer      string `json:"BinaryJailer" mapstructure:"BinaryJailer"`
	ChrootBase        string `json:"ChrootBase" mapstructure:"ChrootBase"`

	JailerGID      int `json:"JailerGid" mapstructure:"JailerGid"`
	JailerNumeNode int `json:"JailerNumaNode" mapstructure:"JailerNumaNode"`
	JailerUID      int `json:"JailerUid" mapstructure:"JailerUid"`

	NetNS string `json:"NetNS" mapstructure:"NetNS"`

	vmmID string
}
