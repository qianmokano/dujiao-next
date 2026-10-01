package jsonmap

import "testing"

func TestScanAcceptsStringSource(t *testing.T) {
	var value JSON
	if err := value.Scan(`{"enabled":true,"name":"通行证"}`); err != nil {
		t.Fatalf("scan string: %v", err)
	}
	if value["enabled"] != true || value["name"] != "通行证" {
		t.Fatalf("scanned = %#v", value)
	}
	var blob JSON
	if err := blob.Scan([]byte(`{"k":1}`)); err != nil || blob["k"] == nil {
		t.Fatalf("blob scan = %#v err=%v", blob, err)
	}
}
