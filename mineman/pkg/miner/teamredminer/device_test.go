package teamredminer

import (
	"encoding/json"
	"testing"
)

func TestParseDevice(t *testing.T) {
	const expectedResult = 2
	var sampleOutput = []byte(`          Team Red Miner version 0.8.4
[2021-12-05 02:08:43] Auto-detected AMD OpenCL platform 0
[2021-12-05 02:08:43] Detected 2 devices, listed in pcie bus id order:
[2021-12-05 02:08:43] Miner Platform OpenCL BusId    Name          Model                     Nr CUs
[2021-12-05 02:08:43] ----- -------- ------ -------- ------------- ------------------------- ------
[2021-12-05 02:08:43]     0        0      1 01:00.0  Ellesmere     Radeon RX 580 Series          36
[2021-12-05 02:08:43]     1        0      0 04:00.0  Fiji          AMD Radeon (TM) R9 Fury S     56
[2021-12-05 02:08:43] Successful clean shutdown.
`)

	gpus, err := parseDevices(sampleOutput)
	if err != nil {
		t.Fatal(err)
	}

	if len(gpus) != expectedResult {
		t.Fatalf("expect %d result, got %d", expectedResult, len(gpus))
	}

	first := gpus[0]
	if first.index != 0 ||
		first.platform != 0 ||
		first.opencl != 1 ||
		first.busId != "01:00.0" ||
		first.name != "Ellesmere" ||
		first.model != "Radeon RX 580 Series" ||
		first.cu != 36 {
		encoded, _ := json.Marshal(first)
		t.Fatalf("expect gpu with index=0, platform=0, opencl=1, busid=01:00.0, name=Ellesmere, model=Radeon RX 580 Series, cu=36, got %s", encoded)
	}

	second := gpus[1]
	if second.index != 1 ||
		second.platform != 0 ||
		second.opencl != 0 ||
		second.busId != "04:00.0" ||
		second.name != "Fiji" ||
		second.model != "AMD Radeon (TM) R9 Fury S" ||
		second.cu != 56 {
		encoded, _ := json.Marshal(second)
		t.Fatalf("expect gpu with index=1, platform=0, opencl=0, busid=04:00.0, name=Fiji, model=AMD Radeon (TM) R9 Fury S, cu=56, got %s", encoded)
	}
}
