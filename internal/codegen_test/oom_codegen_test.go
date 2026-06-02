package cgsample

import (
	"os"
	"testing"
)

// TestOOM_Codegen_HugeArrayHeader pins that generated decoders bound a
// slice/map allocation by the remaining input (the emitted CheckLength gate),
// so a hostile length header cannot trigger a multi-GB make before any element
// is read. Without the gate this OOMs / DoS-es on a dozen bytes.
func TestOOM_Codegen_HugeArrayHeader(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	// A body (no 5-byte stream header → UnmarshalQDF treats src as body):
	//   map8{1}: "tags" -> arr32(0xFFFFFFFF), no elements.
	// Wire tags: tagMap8=0xD5, fixstr|len=0x84, tagArr32=0xD4.
	body := []byte{
		0xD5, 0x01, // map, 1 entry
		0x84, 't', 'a', 'g', 's', // key "tags"
		0xD4, 0xFF, 0xFF, 0xFF, 0xFF, // value: arr32 claiming ~4G elements
	}
	var s Sample
	if _, err := s.UnmarshalQDF(body); err == nil {
		t.Fatal("generated decoder accepted hostile arr32 length without bounds check")
	}
	if len(s.Tags) > 1<<20 {
		t.Fatalf("generated decoder allocated huge slice: %d", len(s.Tags))
	}
}

// TestOOM_Codegen_HugeMapHeader is the map-field counterpart.
func TestOOM_Codegen_HugeMapHeader(t *testing.T) {
	if _, err := os.Stat(generatedFile); err != nil {
		t.Skipf("no generated file %s — run TestGenerate first", generatedFile)
	}
	// map8{1}: "meta" -> map32(0xFFFFFFFF), no entries. tagMap32=0xD7.
	body := []byte{
		0xD5, 0x01,
		0x84, 'm', 'e', 't', 'a',
		0xD7, 0xFF, 0xFF, 0xFF, 0xFF,
	}
	var s Sample
	if _, err := s.UnmarshalQDF(body); err == nil {
		t.Fatal("generated decoder accepted hostile map32 length without bounds check")
	}
	if len(s.Meta) > 1<<20 {
		t.Fatalf("generated decoder allocated huge map: %d", len(s.Meta))
	}
}
