package service

import "testing"

func TestIsValidGeofileName(t *testing.T) {
	service := &ServerService{}
	for _, entry := range defaultGeofileEntries {
		t.Run(entry.FileName, func(t *testing.T) {
			if !service.IsValidGeofileName(entry.FileName) {
				t.Fatalf("valid geofile name %q was rejected", entry.FileName)
			}
		})
	}
	for _, name := range []string{"", "../geoip.dat", "sub/geoip.dat", `sub\geoip.dat`, `geoip\.dat`, "/geoip.dat", "geoip.json", "geoip.dat.exe"} {
		t.Run("invalid_"+name, func(t *testing.T) {
			if service.IsValidGeofileName(name) {
				t.Fatalf("unsafe or invalid geofile name %q was accepted", name)
			}
		})
	}
}
