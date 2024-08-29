package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func main() {
	migGpus, err := GetMigGpus()
	if err != nil {
		log.Fatalf("Failed to get mig GPUs, err: %s", err)
	}

	jsonBytes, _ := json.MarshalIndent(migGpus, "", "  ")
	log.Print(string(jsonBytes))
}

type MigGpu struct {
	ParentGpuId    string
	ParentGpuIndex int

	GpuId      string
	InstanceId int
	MigIndex   int
}

func GetMigGpus() (map[string]MigGpu, error) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to initialize NVML: %v", nvml.ErrorString(ret))
	}
	defer nvml.Shutdown()

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("unable to get device count: %v", nvml.ErrorString(ret))
	}

	result := map[string]MigGpu{}
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("unable to get device at index %d: %v", i, nvml.ErrorString(ret))
		}

		{
			current, _, ret := device.GetMigMode()
			if ret != nvml.SUCCESS {
				return nil, fmt.Errorf("unable to get mig mode at index %d: %v", i, nvml.ErrorString(ret))
			}

			switch current {
			case nvml.DEVICE_MIG_DISABLE:
				continue
			}
		}

		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("unable to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
		}

		migCount, ret := device.GetMaxMigDeviceCount()
		if ret != nvml.SUCCESS {
			return nil, fmt.Errorf("unable to get MIG device count for device at index %d: %v", i, nvml.ErrorString(ret))
		}

		for j := 0; j < migCount; j++ {
			migDevice, ret := device.GetMigDeviceHandleByIndex(j)
			if ret == nvml.ERROR_NOT_FOUND {
				continue
			}
			if ret != nvml.SUCCESS {
				return nil, fmt.Errorf("unable to get MIG device at midx %d for device at pidx %d: %v", j, i, nvml.ErrorString(ret))
			}

			currentMigGpu := MigGpu{
				ParentGpuId:    uuid,
				ParentGpuIndex: i,
				MigIndex:       j,
			}

			{
				migUUID, ret := migDevice.GetUUID()
				if ret != nvml.SUCCESS {
					return nil, fmt.Errorf("unable to get UUID of MIG device at midx %d for device at pidx %d: %v", j, i, nvml.ErrorString(ret))
				}

				currentMigGpu.GpuId = migUUID
			}

			{
				instanceId, ret := migDevice.GetGpuInstanceId()
				if ret != nvml.SUCCESS {
					return nil, fmt.Errorf("unable to get instance id of MIG device at midx %d for device at pidx %d: %v", j, i, nvml.ErrorString(ret))
				}

				currentMigGpu.InstanceId = instanceId
			}

			result[currentMigGpu.GpuId] = currentMigGpu
		}
	}

	return result, nil
}
