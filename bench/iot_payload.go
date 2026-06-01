package bench

import (
	"fmt"
	"math/rand/v2"
)

// DeviceReading holds one device's full sensor batch.
type DeviceReading struct {
	DeviceID string            `json:"device_id" msgpack:"device_id" qdf:"device_id"`
	Ts       []int64           `json:"ts" msgpack:"ts" qdf:"ts"`
	Temp     []float64         `json:"temp" msgpack:"temp" qdf:"temp"`
	Humidity []float64         `json:"humidity" msgpack:"humidity" qdf:"humidity"`
	Tags     map[string]string `json:"tags" msgpack:"tags" qdf:"tags"`
}

// IoTBatch is a batch of DeviceReading values.
type IoTBatch struct {
	Devices []DeviceReading `json:"devices" msgpack:"devices" qdf:"devices"`
}

// mkIoTBatch generates a deterministic IoTBatch with the given number of
// devices and samples per device.
//
// The fixture exercises:
//   - Monotonic int64 timestamps with small jitter → Delta+FOR friendly
//   - Smooth float64 random-walk series → Gorilla/ALP friendly
//   - Low-cardinality map tags (site, fw_version, region) → dict wins
//   - Moderate-cardinality DeviceID strings ("dev-NNNN")
func mkIoTBatch(devices, samples int) IoTBatch {
	rng := rand.New(rand.NewPCG(0xdeadbeef, 0xcafebabe))

	sites := []string{"berlin-dc1", "london-dc2", "tokyo-dc3", "ny-dc4", "sydney-dc5"}
	fwVersions := []string{"fw-1.0.3", "fw-1.1.0", "fw-1.1.2", "fw-2.0.0"}
	regions := []string{"eu-west", "eu-central", "ap-northeast", "us-east", "ap-southeast"}

	baseTs := int64(1_700_000_000_000_000_000) // 2023-11-14 ~22:13 UTC in nanoseconds

	batch := IoTBatch{Devices: make([]DeviceReading, devices)}
	for d := range batch.Devices {
		ts := make([]int64, samples)
		temp := make([]float64, samples)
		humidity := make([]float64, samples)

		// Monotonic timestamps: ~1s step + small jitter (±10ms)
		t := baseTs + int64(d)*int64(1e9)     // stagger start per device
		curTemp := 18.0 + rng.Float64()*12.0  // 18–30 °C
		curHumid := 40.0 + rng.Float64()*30.0 // 40–70 %

		for i := range ts {
			jitter := int64(rng.IntN(20_000_000) - 10_000_000) // ±10ms in ns
			t += 1_000_000_000 + jitter
			ts[i] = t

			// Small random-walk: ±0.1 °C per step
			curTemp += (rng.Float64() - 0.5) * 0.2
			temp[i] = curTemp

			// Small random-walk: ±0.2 % per step
			curHumid += (rng.Float64() - 0.5) * 0.4
			humidity[i] = curHumid
		}

		tags := map[string]string{
			"site":       sites[rng.IntN(len(sites))],
			"fw_version": fwVersions[rng.IntN(len(fwVersions))],
			"region":     regions[rng.IntN(len(regions))],
		}

		batch.Devices[d] = DeviceReading{
			DeviceID: fmt.Sprintf("dev-%04d", d),
			Ts:       ts,
			Temp:     temp,
			Humidity: humidity,
			Tags:     tags,
		}
	}
	return batch
}
