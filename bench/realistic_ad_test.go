package bench

// Realistic Active-Directory-style round-trip test.
//
// This is a *test*, not a benchmark: it verifies that a realistic
// directory-export workload round-trips bit-exactly across every
// encoding option combination, and — as a side effect — reports
// realistic memory/throughput numbers gathered with Go runtime tools
// (runtime.MemStats for allocation churn, syscall.Getrusage for the
// process RSS high-water mark, time for wall-clock throughput).
//
// Run it with:
//
//	go test -v -run TestRealisticAD ./bench
//
// The dataset models an OU dump: every user has high-cardinality unique
// fields (objectGUID, UPN, mail, phone, employeeID), names/surnames that
// repeat occasionally (drawn from syllable pools), and group membership /
// department / title that repeat moderately ("groups, not too frequent").
// That cardinality mix is exactly where Dense/dict/intern codecs earn
// their keep, so the numbers reflect a real directory sync, not a
// synthetic best/worst case.

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"runtime"
	"testing"
	"time"

	qdf "github.com/alex60217101990/qdf"
)

// ADUser mirrors the attributes you would pull from an LDAP/AD user object.
type ADUser struct {
	ObjectGUID  string            `qdf:"objectGUID"`             // unique, high-card
	SAM         string            `qdf:"sAMAccountName"`         // semi-unique
	UPN         string            `qdf:"userPrincipalName"`      // unique
	Email       string            `qdf:"mail"`                   // unique
	FirstName   string            `qdf:"givenName"`              // occasional repeats
	LastName    string            `qdf:"sn"`                     // occasional repeats
	DisplayName string            `qdf:"displayName"`            // occasional repeats
	Department  string            `qdf:"department"`             // moderate repeats
	Title       string            `qdf:"title"`                  // moderate repeats
	Company     string            `qdf:"company"`                // very repeated
	Office      string            `qdf:"physicalDeliveryOffice"` // moderate repeats
	Enabled     bool              `qdf:"enabled"`                // boolean
	LogonCount  int32             `qdf:"logonCount"`             // small int
	WhenCreated int64             `qdf:"whenCreated"`            // timestamp
	LastLogon   int64             `qdf:"lastLogonTimestamp"`     // timestamp
	PwdLastSet  int64             `qdf:"pwdLastSet"`             // timestamp
	MemberOf    []string          `qdf:"memberOf"`               // group DNs, moderate repeats
	Attrs       map[string]string `qdf:"attrs"`                  // stable key-set, mixed-card values
}

// makeADUsers builds a deterministic directory export of n users.
func makeADUsers(n int) []ADUser {
	r := rand.New(rand.NewPCG(0x41445553, 0x65727300)) // "ADUS","ers\0"

	// Name syllable pools -> ~750 first / ~900 last combos => names repeat
	// only occasionally across a few-thousand-user org.
	firstA := []string{"Al", "Bri", "Cas", "Dan", "El", "Fra", "Gab", "Han", "Iv", "Jo",
		"Kar", "Li", "Mar", "Nat", "Ol", "Pa", "Qui", "Ra", "Sa", "Ta",
		"Um", "Vi", "Wen", "Xa", "Yu", "Za", "Be", "Ce", "Do", "Em"}
	firstB := []string{"an", "ana", "el", "ena", "ia", "ian", "ie", "in", "ina", "is",
		"ius", "on", "or", "ra", "ric", "sa", "son", "ta", "us", "var",
		"wen", "ya", "yn", "za", "o"}
	lastA := []string{"Ander", "Bred", "Clark", "Dun", "East", "Fish", "Green", "Hart", "Iver", "Jack",
		"Kels", "Lan", "Mart", "Nor", "Owen", "Park", "Quin", "Reed", "Smith", "Tann",
		"Up", "Vance", "Walk", "Xin", "York", "Zim", "Brook", "Cald", "Drake", "Ellis"}
	lastB := []string{"son", "sen", "field", "ton", "man", "berg", "ford", "wood", "worth", "ley",
		"by", "dale", "stone", "well", "ham", "ric", "gard", "mann", "strom", "ski",
		"ovic", "er", "s", "y", "o", "a", "is", "en", "ino", "ez"}

	departments := []string{"Engineering", "Sales", "Marketing", "Finance", "HR", "Legal",
		"Operations", "Support", "IT", "Security", "Research", "Procurement",
		"Facilities", "QA", "Design", "Data", "Product", "Compliance",
		"Payroll", "Logistics", "PR", "Training", "Audit", "Strategy"}
	titles := []string{"Engineer", "Senior Engineer", "Staff Engineer", "Manager", "Director",
		"VP", "Analyst", "Senior Analyst", "Specialist", "Coordinator",
		"Lead", "Architect", "Consultant", "Associate", "Administrator",
		"Technician", "Representative", "Officer", "Supervisor", "Intern",
		"Principal", "Advisor", "Strategist", "Auditor", "Recruiter",
		"Accountant", "Designer", "Scientist", "Planner", "Clerk",
		"Executive", "Partner"}
	companies := []string{"Contoso", "Contoso EU", "Contoso APAC", "Fabrikam", "Fabrikam Labs"}
	offices := []string{"NYC-1", "NYC-2", "SF-1", "SF-2", "LON-1", "LON-2", "BER-1", "TOK-1",
		"SYD-1", "TOR-1", "CHI-1", "AUS-1", "SEA-1", "BOS-1", "MIA-1", "DEN-1",
		"PAR-1", "AMS-1", "DUB-1", "SGP-1", "HKG-1", "MUM-1", "SAO-1", "MEX-1"}
	countries := []string{"US", "GB", "DE", "FR", "JP", "AU", "CA", "BR", "IN", "SG",
		"NL", "IE", "MX", "ES", "IT", "SE", "PL", "CH", "AE", "ZA"}
	cities := []string{"New York", "San Francisco", "London", "Berlin", "Tokyo", "Sydney",
		"Toronto", "Chicago", "Austin", "Seattle", "Boston", "Miami",
		"Denver", "Paris", "Amsterdam", "Dublin", "Singapore", "Mumbai",
		"Sao Paulo", "Mexico City", "Madrid", "Milan", "Stockholm", "Zurich"}

	// Group DN pool: 400 distinct security groups -> moderate membership overlap.
	const nGroups = 400
	groups := make([]string, nGroups)
	for i := range groups {
		groups[i] = fmt.Sprintf("CN=SG-%03d,OU=Groups,DC=corp,DC=contoso,DC=com", i)
	}
	// Manager DN pool: 120 managers.
	const nMgr = 120
	managers := make([]string, nMgr)
	for i := range managers {
		managers[i] = fmt.Sprintf("CN=Mgr-%03d,OU=Users,DC=corp,DC=contoso,DC=com", i)
	}

	hexd := "0123456789abcdef"
	hex32 := func() string {
		b := make([]byte, 32)
		for i := range b {
			b[i] = hexd[r.IntN(16)]
		}
		return string(b)
	}

	// Plausible AD epoch range (~2018..2024 in seconds).
	const tBase = 1514764800 // 2018-01-01
	const tSpan = 220752000  // ~7 years

	out := make([]ADUser, n)
	for i := range out {
		first := firstA[r.IntN(len(firstA))] + firstB[r.IntN(len(firstB))]
		last := lastA[r.IntN(len(lastA))] + lastB[r.IntN(len(lastB))]
		sam := fmt.Sprintf("%c%s%d", first[0]|0x20, lowerASCII(last), i%10000)
		upn := sam + "@corp.contoso.com"

		created := int64(tBase + r.IntN(tSpan))
		lastLogon := created + int64(r.IntN(tSpan))
		pwdSet := created + int64(r.IntN(int(lastLogon-created)+1))

		// Membership: 2..6 groups.
		k := 2 + r.IntN(5)
		mo := make([]string, k)
		for j := range mo {
			mo[j] = groups[r.IntN(nGroups)]
		}

		attrs := map[string]string{
			"employeeID":      fmt.Sprintf("E%06d", r.IntN(900000)+100000),
			"costCenter":      fmt.Sprintf("CC-%03d", r.IntN(60)),
			"manager":         managers[r.IntN(nMgr)],
			"co":              countries[r.IntN(len(countries))],
			"l":               cities[r.IntN(len(cities))],
			"telephoneNumber": fmt.Sprintf("+1-%03d-555-%04d", r.IntN(900)+100, r.IntN(10000)),
		}

		out[i] = ADUser{
			ObjectGUID:  hex32(),
			SAM:         sam,
			UPN:         upn,
			Email:       upn,
			FirstName:   first,
			LastName:    last,
			DisplayName: first + " " + last,
			Department:  departments[r.IntN(len(departments))],
			Title:       titles[r.IntN(len(titles))],
			Company:     companies[r.IntN(len(companies))],
			Office:      offices[r.IntN(len(offices))],
			Enabled:     r.IntN(100) < 92, // ~8% disabled
			LogonCount:  int32(r.IntN(5000)),
			WhenCreated: created,
			LastLogon:   lastLogon,
			PwdLastSet:  pwdSet,
			MemberOf:    mo,
			Attrs:       attrs,
		}
	}
	return out
}

func lowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] |= 0x20
		}
	}
	return string(b)
}

// phaseStat holds the runtime-measured numbers for one encode or decode phase.
type phaseStat struct {
	wire       int
	nsPerOp    float64
	allocPerOp uint64 // bytes allocated per op (runtime.MemStats.TotalAlloc delta)
	objsPerOp  uint64 // allocations per op (runtime.MemStats.Mallocs delta)
}

// datasetCardinality reports distinct-value counts so the test output proves
// the dataset has the intended cardinality mix (unique IDs, occasional name
// repeats, moderate group/department repeats) rather than a degenerate one.
func datasetCardinality(users []ADUser) string {
	guid := map[string]struct{}{}
	first := map[string]struct{}{}
	last := map[string]struct{}{}
	dept := map[string]struct{}{}
	grp := map[string]struct{}{}
	for i := range users {
		u := &users[i]
		guid[u.ObjectGUID] = struct{}{}
		first[u.FirstName] = struct{}{}
		last[u.LastName] = struct{}{}
		dept[u.Department] = struct{}{}
		for _, g := range u.MemberOf {
			grp[g] = struct{}{}
		}
	}
	n := float64(len(users))
	return fmt.Sprintf("distinct: guid=%d(%.2f/user) first=%d last=%d dept=%d group=%d",
		len(guid), float64(len(guid))/n, len(first), len(last), len(dept), len(grp))
}

// RSS high-water is read via the shared maxRSSBytes helper
// (memory_compare_test.go) — syscall.Getrusage, no cgo, no deps.

func measureEncode(t *testing.T, v any, opt qdf.Options, iters int) (phaseStat, []byte) {
	t.Helper()
	b, err := qdf.Marshal(v, opt) // warm up + size
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	start := time.Now()
	for i := range iters {
		b, err = qdf.Marshal(v, opt)
		if err != nil {
			t.Fatalf("encode iter %d: %v", i, err)
		}
	}
	el := time.Since(start)
	runtime.ReadMemStats(&m1)

	s := phaseStat{
		wire:       len(b),
		nsPerOp:    float64(el.Nanoseconds()) / float64(iters),
		allocPerOp: (m1.TotalAlloc - m0.TotalAlloc) / uint64(iters),
		objsPerOp:  (m1.Mallocs - m0.Mallocs) / uint64(iters),
	}
	return s, b
}

func measureDecode[T any](t *testing.T, data []byte, iters int) (phaseStat, T) {
	t.Helper()
	var out T
	if err := qdf.Unmarshal(data, &out); err != nil { // warm up
		t.Fatalf("decode: %v", err)
	}

	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	start := time.Now()
	for i := range iters {
		var d T
		if err := qdf.Unmarshal(data, &d); err != nil {
			t.Fatalf("decode iter %d: %v", i, err)
		}
		out = d
	}
	el := time.Since(start)
	runtime.ReadMemStats(&m1)

	s := phaseStat{
		wire:       len(data),
		nsPerOp:    float64(el.Nanoseconds()) / float64(iters),
		allocPerOp: (m1.TotalAlloc - m0.TotalAlloc) / uint64(iters),
		objsPerOp:  (m1.Mallocs - m0.Mallocs) / uint64(iters),
	}
	return s, out
}

func TestRealisticAD_EncodeDecode(t *testing.T) {
	n := 5000
	iters := 12
	if testing.Short() {
		n, iters = 800, 3
	}
	users := makeADUsers(n)

	combos := []struct {
		name string
		opt  qdf.Options
	}{
		{"Speed", qdf.OptSpeed},
		{"Balanced", qdf.OptBalanced},
		{"Balanced+MapShape", qdf.OptBalanced | qdf.OptMapShape},
		{"Balanced+ColumnIndex", qdf.OptBalanced | qdf.OptColumnIndex},
		{"Compression", qdf.OptCompression},
		{"Compression+MapShape+ColIdx", qdf.OptCompression | qdf.OptMapShape | qdf.OptColumnIndex},
	}

	// Reference logical size: the OptSpeed wire is the codec-free serialization
	// of the dataset — the same logical bytes every mode must traverse. Using it
	// (not each mode's own compressed wire) as the throughput denominator keeps
	// the MB/s metric apples-to-apples: a mode that compresses harder produces
	// fewer wire bytes but does NOT process less logical data, so dividing by its
	// own shrunken wire would falsely report it as "slower".
	refBlob, err := qdf.Marshal(users, qdf.OptSpeed)
	if err != nil {
		t.Fatalf("reference encode: %v", err)
	}
	refMB := float64(len(refBlob)) / (1 << 20)
	mbps := func(ns float64) float64 { return refMB / (ns / 1e9) }
	mrecs := func(ns float64) float64 { return float64(n) / (ns / 1e9) / 1e6 }

	rssStart := maxRSSBytes()
	t.Logf("AD directory export: %d users, iters=%d (GOOS=%s GOMAXPROCS=%d)",
		n, iters, runtime.GOOS, runtime.GOMAXPROCS(0))
	t.Logf("  %s", datasetCardinality(users))
	t.Logf("  reference logical size (OptSpeed wire) = %.2f MB; throughput MB/s is vs this fixed size",
		refMB)
	t.Logf("%-30s %8s %6s | %8s %8s %8s %7s | %8s %8s %8s %7s",
		"opt-combo", "wireKB", "ratio",
		"encMrec/s", "enc_MB/s", "enc_KB", "enc_obj",
		"decMrec/s", "dec_MB/s", "dec_KB", "dec_obj")

	for _, c := range combos {
		es, blob := measureEncode(t, users, c.opt, iters)
		ds, decoded := measureDecode[[]ADUser](t, blob, iters)

		if !reflect.DeepEqual(users, decoded) {
			t.Fatalf("round-trip mismatch for %q (wire=%d bytes)", c.name, es.wire)
		}

		t.Logf("%-30s %8.1f %6.2f | %8.2f %8.1f %8.1f %7d | %8.2f %8.1f %8.1f %7d",
			c.name,
			float64(es.wire)/1024, float64(len(refBlob))/float64(es.wire),
			mrecs(es.nsPerOp), mbps(es.nsPerOp), float64(es.allocPerOp)/1024, es.objsPerOp,
			mrecs(ds.nsPerOp), mbps(ds.nsPerOp), float64(ds.allocPerOp)/1024, ds.objsPerOp,
		)
	}

	rssEnd := maxRSSBytes()
	t.Logf("process RSS (Getrusage Maxrss high-water): start=%.1f MB  end=%.1f MB  growth=%.1f MB",
		float64(rssStart)/(1<<20), float64(rssEnd)/(1<<20),
		(float64(rssEnd)-float64(rssStart))/(1<<20))
	t.Logf("note: enc_KB/dec_KB = bytes allocated per op (TotalAlloc delta); " +
		"RSS is a process-wide high-water mark, monotonic across phases")
}

// ADUserFlat is ADUser with the two columnar-disqualifying fields removed
// (MemberOf []string, Attrs map[string]string). Every remaining field is a
// scalar / string / timestamp, so buildColumnarPlan accepts it and the
// []ADUserFlat batch encodes via the columnar transpose. This is the headroom
// probe for hybrid columnar: the delta between the full row-major struct and
// this flat columnar struct, on the same scalar/string data, is what a hybrid
// (transpose eligible columns, keep map/slice row-wise) would unlock.
type ADUserFlat struct {
	ObjectGUID  string `qdf:"objectGUID"`
	SAM         string `qdf:"sAMAccountName"`
	UPN         string `qdf:"userPrincipalName"`
	Email       string `qdf:"mail"`
	FirstName   string `qdf:"givenName"`
	LastName    string `qdf:"sn"`
	DisplayName string `qdf:"displayName"`
	Department  string `qdf:"department"`
	Title       string `qdf:"title"`
	Company     string `qdf:"company"`
	Office      string `qdf:"physicalDeliveryOffice"`
	Enabled     bool   `qdf:"enabled"`
	LogonCount  int32  `qdf:"logonCount"`
	WhenCreated int64  `qdf:"whenCreated"`
	LastLogon   int64  `qdf:"lastLogonTimestamp"`
	PwdLastSet  int64  `qdf:"pwdLastSet"`
}

func flattenAD(users []ADUser) []ADUserFlat {
	out := make([]ADUserFlat, len(users))
	for i := range users {
		u := &users[i]
		out[i] = ADUserFlat{
			ObjectGUID: u.ObjectGUID, SAM: u.SAM, UPN: u.UPN, Email: u.Email,
			FirstName: u.FirstName, LastName: u.LastName, DisplayName: u.DisplayName,
			Department: u.Department, Title: u.Title, Company: u.Company, Office: u.Office,
			Enabled: u.Enabled, LogonCount: u.LogonCount,
			WhenCreated: u.WhenCreated, LastLogon: u.LastLogon, PwdLastSet: u.PwdLastSet,
		}
	}
	return out
}

// TestRealisticAD_FlatColumnar_Headroom measures the columnar path on the
// flat AD struct and prints it next to the full row-major struct, so the gain
// hybrid columnar would bring to AD-shaped data is visible. It also ASSERTS
// the columnar transpose actually engaged (OptColumnIndex changes the wire
// only for columnar payloads; if the wire is identical, columnar did not fire
// and the probe is meaningless).
func TestRealisticAD_FlatColumnar_Headroom(t *testing.T) {
	n := 5000
	iters := 12
	if testing.Short() {
		n, iters = 800, 3
	}
	full := makeADUsers(n)
	flat := flattenAD(full)

	// Prove columnar engaged for the flat struct: the column-length index is
	// emitted only on a columnar payload, so it must change the wire size.
	noIdx, err := qdf.Marshal(flat, qdf.OptBalanced)
	if err != nil {
		t.Fatalf("encode flat: %v", err)
	}
	withIdx, err := qdf.Marshal(flat, qdf.OptBalanced|qdf.OptColumnIndex)
	if err != nil {
		t.Fatalf("encode flat+idx: %v", err)
	}
	columnar := len(noIdx) != len(withIdx)
	t.Logf("flat columnar engaged: %v (Balanced wire=%d, +ColumnIndex wire=%d)",
		columnar, len(noIdx), len(withIdx))
	if !columnar {
		t.Fatalf("expected []ADUserFlat to encode columnar (ColumnIndex changed nothing) — "+
			"buildColumnarPlan or columnarProbe rejected it; wire=%d", len(noIdx))
	}

	refFlat, _ := qdf.Marshal(flat, qdf.OptSpeed)
	refMB := float64(len(refFlat)) / (1 << 20)
	mbps := func(ns float64) float64 { return refMB / (ns / 1e9) }

	combos := []struct {
		name string
		opt  qdf.Options
	}{
		{"Speed", qdf.OptSpeed},
		{"Balanced", qdf.OptBalanced},
		{"Balanced+ColumnIndex", qdf.OptBalanced | qdf.OptColumnIndex},
		{"Compression", qdf.OptCompression},
	}

	t.Logf("FLAT (columnar-eligible) AD: %d users, ref logical=%.2f MB", n, refMB)
	t.Logf("%-24s %8s %6s | %8s %8s %7s | %8s %8s %7s",
		"opt-combo", "wireKB", "ratio", "enc_MB/s", "enc_KB", "enc_obj",
		"dec_MB/s", "dec_KB", "dec_obj")
	for _, c := range combos {
		es, blob := measureEncode(t, flat, c.opt, iters)
		ds, decoded := measureDecode[[]ADUserFlat](t, blob, iters)
		if !reflect.DeepEqual(flat, decoded) {
			t.Fatalf("flat round-trip mismatch for %q", c.name)
		}
		t.Logf("%-24s %8.1f %6.2f | %8.1f %8.1f %7d | %8.1f %8.1f %7d",
			c.name, float64(es.wire)/1024, float64(len(refFlat))/float64(es.wire),
			mbps(es.nsPerOp), float64(es.allocPerOp)/1024, es.objsPerOp,
			mbps(ds.nsPerOp), float64(ds.allocPerOp)/1024, ds.objsPerOp)
	}

	// Hybrid-columnar headroom estimate at Balanced.
	//
	//   full   = all 17 fields, row-major (today's behaviour for AD)
	//   flat   = 15 scalar/string fields, columnar
	//   resid  = the 2 disqualifying fields (MemberOf []string, Attrs map),
	//            row-major — what hybrid would still encode per-row
	//
	// hybrid ≈ flat (columnar columns) + resid (row-major tail). Comparing
	// that estimate to `full` isolates the columnar win from the mere removal
	// of the map/slice fields.
	resid := make([]ADUserResidual, n)
	for i := range full {
		resid[i] = ADUserResidual{MemberOf: full[i].MemberOf, Attrs: full[i].Attrs}
	}

	fullDec, _ := measureDecode[[]ADUser](t, mustMarshal(t, full, qdf.OptBalanced), iters)
	flatDec, _ := measureDecode[[]ADUserFlat](t, mustMarshal(t, flat, qdf.OptBalanced), iters)
	residDec, _ := measureDecode[[]ADUserResidual](t, mustMarshal(t, resid, qdf.OptBalanced), iters)

	hybridObj := flatDec.objsPerOp + residDec.objsPerOp
	hybridKB := float64(flatDec.allocPerOp+residDec.allocPerOp) / 1024
	t.Logf("Balanced decode head-to-head (isolating columnar from map/slice removal):")
	t.Logf("  FULL row-major (today) : %d obj / %.0f KB", fullDec.objsPerOp, float64(fullDec.allocPerOp)/1024)
	t.Logf("  FLAT columnar (15 cols): %d obj / %.0f KB", flatDec.objsPerOp, float64(flatDec.allocPerOp)/1024)
	t.Logf("  RESID row-major (2 fld): %d obj / %.0f KB", residDec.objsPerOp, float64(residDec.allocPerOp)/1024)
	t.Logf("  HYBRID est (flat+resid): %d obj / %.0f KB  => %.1fx fewer decode allocs vs full",
		hybridObj, hybridKB, float64(fullDec.objsPerOp)/float64(maxU(hybridObj, 1)))
}

// ADUserResidual holds only the columnar-disqualifying fields of ADUser — the
// per-row tail a hybrid columnar encoder would keep row-major.
type ADUserResidual struct {
	MemberOf []string          `qdf:"memberOf"`
	Attrs    map[string]string `qdf:"attrs"`
}

func mustMarshal(t *testing.T, v any, opt qdf.Options) []byte {
	t.Helper()
	b, err := qdf.Marshal(v, opt)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func maxU(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
