package kekahu

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bbengfort/x/unique"
	"github.com/opalmer/check-go-version/api"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// HealthCheck returns the system status, fetching all components of the status.
// Note that fetching system information can fail in several places, all
// status compenents are attempted, then aggregated into a single error message,
// which means a partially populated stuct can be returned. If ignoreErrors
// is true, then no error will be returned unless ALL status components fail
// and a completely empty struct is returned. If it is false, then if any one
// status component fails, that error is returned immediately.
//
// It is recommended to call this function with ignoreErrors=true
func HealthCheck(ignoreErrors bool) (status *SystemStatus, err error) {
	// Create the system status and call all status component checks
	status = new(SystemStatus)

	// Status components to call to populate the system information.
	statusComponents := []func() error{
		status.getHostStatus,
		status.getMemStatus,
		status.getDiskStatus,
		status.getCPUStatus,
		status.getUtilizationStatus,
		status.getGoRuntime,
	}

	// Keep track of the errors from each status component
	statusErrors := make([]error, 0, len(statusComponents))

	// Call each status component and check for errors
	for _, check := range statusComponents {
		if err = check(); err != nil {
			statusErrors = append(statusErrors, err)
			if !ignoreErrors {
				return nil, err
			}
		}
	}

	// Return an error if all status components failed
	if len(statusErrors) == len(statusComponents) {
		return nil, errors.New("all status component checks failed, for more details set ignoreErrors=False")
	}

	return status, nil
}

// SystemStatus provides a simple machine health status report, implemented
// from github.com/rebeccabilbro/doctor. It contains OS and Go version and
// platform information as well as information about system resources such as
// disk, memory, and CPU.
type SystemStatus struct {
	Hostname        string  `json:"hostname,omitempty"`          // hostname identified by OS
	OS              string  `json:"os,omitempty"`                // operating system name, e.g. darwin, linux
	Platform        string  `json:"platform,omitempty"`          // specific os version e.g. ubuntu, linuxmint
	PlatformVersion string  `json:"platform_version,omitempty"`  // operating system version number
	ActiveProcesses uint64  `json:"active_procs,omitempty"`      // number of active processes
	Uptime          uint64  `json:"uptime,omitempty"`            // number of seconds the host has been online
	TotalRAM        uint64  `json:"total_ram,omitempty"`         // total amount of RAM on the system
	AvailableRAM    uint64  `json:"available_ram,omitempty"`     // RAM available for programs to allocate (from kernel)
	UsedRAM         uint64  `json:"used_ram,omitempty"`          // amount of RAM used by programs (from kernel)
	UsedRAMPercent  float64 `json:"used_ram_percent,omitempty"`  // percentage of RAM used by programs
	Filesystem      string  `json:"filesystem,omitempty"`        // the type of filesystem at root
	TotalDisk       uint64  `json:"total_disk,omitempty"`        // total amount of disk space available at root directory
	FreeDisk        uint64  `json:"free_disk,omitempty"`         // total amount of unused disk space at root directory
	UsedDisk        uint64  `json:"used_disk,omitempty"`         // total amount of disk space used by root directory
	UsedDiskPercent float64 `json:"used_disk_percent,omitempty"` // percentage of disk space used by root directory
	CPUModel        string  `json:"cpu_model,omitempty"`         // the model of CPU on the machine
	CPUCores        int32   `json:"cpu_cores,omitempty"`         // the number of CPU cores detected
	CPUPercent      float64 `json:"cpu_percent,omitempty"`       // the percentage of all cores being used over the last 5 seconds
	GoVersion       string  `json:"go_version,omitempty"`        // the version of Go for the currently running instance
	GoPlatform      string  `json:"go_platform,omitempty"`       // the platform compiled for the currently running instance
	GoArchitecture  string  `json:"go_architecture,omitempty"`   // the chip architecture compiled for the currently running instance
}

// Dump the system status to JSON with the specified indent
func (s *SystemStatus) Dump(indent int) (data []byte, err error) {
	if indent == 0 {
		return json.Marshal(s)
	}

	sindent := strings.Repeat(" ", indent)
	return json.MarshalIndent(s, "", sindent)
}

// Get the host info elements of the status
func (s *SystemStatus) getHostStatus() (err error) {
	// Get the host information
	var info *host.InfoStat
	if info, err = host.Info(); err != nil {
		return err
	}

	// Populate the status with the host info
	s.Hostname = info.Hostname
	s.OS = info.OS
	s.Platform = info.Platform
	s.PlatformVersion = info.PlatformVersion
	s.ActiveProcesses = info.Procs
	s.Uptime = info.Uptime

	return nil
}

// Get the memory info elements of the status
func (s *SystemStatus) getMemStatus() (err error) {
	// Get the memory information
	var info *mem.VirtualMemoryStat
	if info, err = mem.VirtualMemory(); err != nil {
		return err
	}

	//Populate the status with memory info
	s.TotalRAM = info.Total
	s.AvailableRAM = info.Available
	s.UsedRAM = info.Used
	s.UsedRAMPercent = info.UsedPercent

	return nil
}

// Get the disk info elements of the status
// TODO: pass in the path to this function
func (s *SystemStatus) getDiskStatus() (err error) {
	// Get the memory information
	var info *disk.UsageStat
	if info, err = disk.Usage("/"); err != nil {
		return err
	}

	//Populate the status with memory info
	s.Filesystem = info.Fstype
	s.TotalDisk = info.Total
	s.FreeDisk = info.Free
	s.UsedDisk = info.Used
	s.UsedDiskPercent = info.UsedPercent

	return nil
}

// Get the CPU info elements of the status
func (s *SystemStatus) getCPUStatus() (err error) {
	// Get the cpu information
	var info []cpu.InfoStat
	if info, err = cpu.Info(); err != nil {
		return err
	}

	// All CPU model names detected
	names := make([]string, 0, len(info))

	// Populate the status with memory info
	for _, i := range info {
		names = append(names, i.ModelName)
		s.CPUCores += i.Cores
	}

	// Get only the unique CPU model names
	names = unique.Strings(names)
	s.CPUModel = strings.Join(names, ", ")

	return nil
}

// Get the CPU percent utilization element of the status
// TODO: pass in the duration to this function
func (s *SystemStatus) getUtilizationStatus() (err error) {
	// Get utilization information
	var info []float64
	if info, err = cpu.Percent(time.Second*5, false); err != nil {
		return err
	}

	// Populate status with utilization info
	// Note that percpu is false, so only one percentage is returned
	s.CPUPercent = info[0]

	return nil
}

// Get the Go runtime version information
func (s *SystemStatus) getGoRuntime() (err error) {
	// Get runtime information
	var info *api.Version
	if info, err = api.GetRunningVersion(); err != nil {
		return err
	}

	// Populate the status with runtime info
	s.GoVersion = info.FullVersion
	s.GoPlatform = info.Platform
	s.GoArchitecture = info.Architecture

	return nil
}
