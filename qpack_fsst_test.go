package qdf

import "testing"

type fsstRow struct {
	Msg string
}

func TestFSSTColumnRoundTrip(t *testing.T) {
	rows := make([]fsstRow, 0, 512)
	paths := []string{
		"GET /api/v1/users/42/profile HTTP/1.1 200",
		"POST /api/v1/orders HTTP/1.1 201",
		"GET /api/v1/users/7/settings HTTP/1.1 404",
	}
	for i := 0; i < 512; i++ {
		rows = append(rows, fsstRow{Msg: paths[i%len(paths)] + " req=" + itoaTiny(i)})
	}
	b, err := Marshal(rows, OptCompression)
	if err != nil {
		t.Fatal(err)
	}
	if !containsByte(b, tagColStrFSST) {
		t.Fatalf("expected FSST tag 0x%X in wire", tagColStrFSST)
	}
	var back []fsstRow
	if err := Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != len(rows) {
		t.Fatalf("len %d want %d", len(back), len(rows))
	}
	for i := range rows {
		if back[i].Msg != rows[i].Msg {
			t.Fatalf("row %d: %q != %q", i, back[i].Msg, rows[i].Msg)
		}
	}
}

func itoaTiny(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}

