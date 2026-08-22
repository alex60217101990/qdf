package bench

import (
	"fmt"
	"math/rand/v2"
)

// Geo holds geographic info for a device.
type Geo struct {
	Country string  `json:"country" msgpack:"country" qdf:"country"`
	Lat     float64 `json:"lat" msgpack:"lat" qdf:"lat"`
	Lon     float64 `json:"lon" msgpack:"lon" qdf:"lon"`
	Type    int     `json:"type" msgpack:"type" qdf:"type"`
}

// Device describes the user's device in a bid request.
type Device struct {
	UA   string `json:"ua" msgpack:"ua" qdf:"ua"`
	IP   string `json:"ip" msgpack:"ip" qdf:"ip"`
	OS   int    `json:"os" msgpack:"os" qdf:"os"`
	Type int    `json:"type" msgpack:"type" qdf:"type"`
	Geo  Geo    `json:"geo" msgpack:"geo" qdf:"geo"`
}

// Impression represents a single ad slot within a bid request.
type Impression struct {
	ID       string            `json:"id" msgpack:"id" qdf:"id"`
	BidFloor float64           `json:"bid_floor" msgpack:"bid_floor" qdf:"bid_floor"`
	W        int               `json:"w" msgpack:"w" qdf:"w"`
	H        int               `json:"h" msgpack:"h" qdf:"h"`
	DealIDs  []string          `json:"deal_ids" msgpack:"deal_ids" qdf:"deal_ids"`
	Ext      map[string]string `json:"ext" msgpack:"ext" qdf:"ext"`
}

// BidRequest is an OpenRTB-style bid request.
type BidRequest struct {
	ID   string       `json:"id" msgpack:"id" qdf:"id"`
	At   int          `json:"at" msgpack:"at" qdf:"at"`
	Tmax int          `json:"tmax" msgpack:"tmax" qdf:"tmax"`
	Imp  []Impression `json:"imp" msgpack:"imp" qdf:"imp"`
	Dev  Device       `json:"dev" msgpack:"dev" qdf:"dev"`
	Cur  []string     `json:"cur" msgpack:"cur" qdf:"cur"`
}

// mkRTBBatch generates a deterministic batch of n OpenRTB-style bid requests.
// The fixture exercises: repeated enums (dict/RLE wins), high-cardinality hex
// IDs (raw), nested structs, maps, and float columns.
func mkRTBBatch(n int) []BidRequest {
	rng := rand.New(rand.NewPCG(42, 7))

	countries := []string{
		"US", "GB", "DE", "FR", "JP", "CN", "BR", "IN",
		"AU", "CA", "ES", "IT", "MX", "KR", "RU", "NL",
		"SE", "CH", "PL", "AR",
	}
	currencies := []string{"USD", "EUR", "GBP"}
	adSizes := [][2]int{
		{300, 250},
		{728, 90},
		{320, 50},
		{160, 600},
		{300, 600},
		{970, 250},
		{320, 480},
		{300, 50},
	}
	extKeys := []string{"sstype", "ptype", "schain", "dsa", "cattax"}
	extVals := []string{"web", "app", "display", "video", "native", "banner"}

	out := make([]BidRequest, n)
	for i := range out {
		nImp := 1 + rng.IntN(4) // 1–4 impressions

		imps := make([]Impression, nImp)
		for j := range imps {
			sz := adSizes[rng.IntN(len(adSizes))]

			nDeals := rng.IntN(3) // 0–2 deal IDs
			deals := make([]string, nDeals)
			for k := range deals {
				deals[k] = rtbHex(rng, 16)
			}

			nExt := 2 + rng.IntN(2) // 2–3 ext keys
			ext := make(map[string]string, nExt)
			for range nExt {
				ext[extKeys[rng.IntN(len(extKeys))]] = extVals[rng.IntN(len(extVals))]
			}

			imps[j] = Impression{
				ID:       rtbHex(rng, 16),
				BidFloor: float64(rng.IntN(10000)) * 0.0001, // 0.0000–0.9999
				W:        sz[0],
				H:        sz[1],
				DealIDs:  deals,
				Ext:      ext,
			}
		}

		nCur := 1 + rng.IntN(len(currencies))
		cur := make([]string, nCur)
		for k := range cur {
			cur[k] = currencies[rng.IntN(len(currencies))]
		}

		out[i] = BidRequest{
			ID:   rtbHex(rng, 16),
			At:   1 + rng.IntN(2),     // first/second price: 1 or 2
			Tmax: 100 + rng.IntN(150), // 100–249 ms
			Imp:  imps,
			Dev: Device{
				UA:   fmt.Sprintf("Mozilla/5.0 (%s) rv:%d", countries[rng.IntN(len(countries))], 80+rng.IntN(40)),
				IP:   fmt.Sprintf("%d.%d.%d.%d", rng.IntN(256), rng.IntN(256), rng.IntN(256), rng.IntN(256)),
				OS:   1 + rng.IntN(6), // 1–6
				Type: 1 + rng.IntN(7), // 1–7
				Geo: Geo{
					Country: countries[rng.IntN(len(countries))],
					Lat:     float64(rng.IntN(18000)-9000) / 100.0,  // -90..90
					Lon:     float64(rng.IntN(36000)-18000) / 100.0, // -180..180
					Type:    rng.IntN(3),                            // 0–2
				},
			},
			Cur: cur,
		}
	}
	return out
}

// rtbHex returns a deterministic n-char lowercase hex string.
func rtbHex(rng *rand.Rand, n int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hexChars[rng.IntN(16)]
	}
	return string(b)
}
