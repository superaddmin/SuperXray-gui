package service

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xtls/xray-core/common/geodata"
	"google.golang.org/protobuf/proto"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type interruptedResponseBody struct {
	sent bool
}

func (b *interruptedResponseBody) Read(p []byte) (int, error) {
	if b.sent {
		return 0, io.ErrUnexpectedEOF
	}
	b.sent = true
	return copy(p, []byte("partial geofile payload")), nil
}

func (b *interruptedResponseBody) Close() error { return nil }

func mustGeoIPData(t *testing.T, code string) []byte {
	t.Helper()
	data, err := proto.Marshal(&geodata.GeoIPList{Entry: []*geodata.GeoIP{{Code: code}}})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustGeositeData(t *testing.T, code string) []byte {
	t.Helper()
	data, err := proto.Marshal(&geodata.GeoSiteList{Entry: []*geodata.GeoSite{{Code: code}}})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestServerServiceUpdateGeofileInterruptedDownloadPreservesExistingFile(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("XUI_BIN_FOLDER", binDir)
	destPath := filepath.Join(binDir, "geoip.dat")
	original := mustGeoIPData(t, "OLD")
	if err := os.WriteFile(destPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	originalClient := serviceHTTPClient
	serviceHTTPClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Body:          &interruptedResponseBody{},
			Header:        make(http.Header),
			ContentLength: -1,
		}, nil
	})}
	t.Cleanup(func() { serviceHTTPClient = originalClient })

	service := &ServerService{}
	if err := service.UpdateGeofile("geoip.dat"); err == nil {
		t.Fatal("expected interrupted download to fail")
	}
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("existing geofile was removed: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing geofile changed after interrupted download: got %q", got)
	}
}

func TestGeofileUpdateValidationFailurePreservesExistingFileWithoutRestart(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>not geodata</html>"))
	}))
	t.Cleanup(testServer.Close)

	binDir := t.TempDir()
	destPath := filepath.Join(binDir, "geoip.dat")
	original := []byte("existing geofile remains available")
	if err := os.WriteFile(destPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	restartCalls := 0
	updater := geofileUpdater{
		client:  testServer.Client(),
		binDir:  binDir,
		entries: []geofileEntry{{URL: testServer.URL, FileName: "geoip.dat"}},
		restart: func() error {
			restartCalls++
			return nil
		},
	}

	if err := updater.update("geoip.dat"); err == nil {
		t.Fatal("expected invalid geofile to be rejected")
	}
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing geofile changed after validation failure: got %q", got)
	}
	if restartCalls != 0 {
		t.Fatalf("restart calls = %d, want 0", restartCalls)
	}
}

func TestGeofileUpdateNotModifiedDoesNotRestart(t *testing.T) {
	conditionalRequest := false
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conditionalRequest = r.Header.Get("If-Modified-Since") != ""
		w.WriteHeader(http.StatusNotModified)
	}))
	t.Cleanup(testServer.Close)

	binDir := t.TempDir()
	destPath := filepath.Join(binDir, "geoip.dat")
	original := mustGeoIPData(t, "OLD")
	if err := os.WriteFile(destPath, original, 0o600); err != nil {
		t.Fatal(err)
	}

	restartCalls := 0
	updater := geofileUpdater{
		client:  testServer.Client(),
		binDir:  binDir,
		entries: []geofileEntry{{URL: testServer.URL, FileName: "geoip.dat"}},
		restart: func() error {
			restartCalls++
			return nil
		},
	}

	if err := updater.update("geoip.dat"); err != nil {
		t.Fatalf("304 update returned error: %v", err)
	}
	if !conditionalRequest {
		t.Fatal("expected If-Modified-Since for an existing valid geofile")
	}
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatal("304 response changed the existing geofile")
	}
	if restartCalls != 0 {
		t.Fatalf("restart calls = %d, want 0", restartCalls)
	}
}

func TestGeofileUpdateBatchFailurePreservesAllExistingFilesWithoutRestart(t *testing.T) {
	requestCount := 0
	newGeoIP := mustGeoIPData(t, "NEW")
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch r.URL.Path {
		case "/geoip":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(newGeoIP)
		case "/geosite":
			w.WriteHeader(http.StatusBadGateway)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(testServer.Close)

	binDir := t.TempDir()
	oldGeoIP := mustGeoIPData(t, "OLD-IP")
	oldGeosite := mustGeositeData(t, "OLD-SITE")
	geoIPPath := filepath.Join(binDir, "geoip.dat")
	geositePath := filepath.Join(binDir, "geosite.dat")
	if err := os.WriteFile(geoIPPath, oldGeoIP, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(geositePath, oldGeosite, 0o600); err != nil {
		t.Fatal(err)
	}

	restartCalls := 0
	updater := geofileUpdater{
		client: testServer.Client(),
		binDir: binDir,
		entries: []geofileEntry{
			{URL: testServer.URL + "/geoip", FileName: "geoip.dat"},
			{URL: testServer.URL + "/geosite", FileName: "geosite.dat"},
		},
		restart: func() error {
			restartCalls++
			return nil
		},
	}

	if err := updater.update(""); err == nil {
		t.Fatal("expected batch update to fail")
	}
	if requestCount != 2 {
		t.Fatalf("requests = %d, want 2", requestCount)
	}
	gotGeoIP, err := os.ReadFile(geoIPPath)
	if err != nil {
		t.Fatal(err)
	}
	gotGeosite, err := os.ReadFile(geositePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotGeoIP, oldGeoIP) || !bytes.Equal(gotGeosite, oldGeosite) {
		t.Fatal("batch failure changed an existing geofile")
	}
	if restartCalls != 0 {
		t.Fatalf("restart calls = %d, want 0", restartCalls)
	}
}

func TestGeofileUpdateChangedAndNotModifiedRestartsOnce(t *testing.T) {
	newGeoIP := mustGeoIPData(t, "NEW-IP")
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/geoip":
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(newGeoIP)
		case "/geosite":
			w.WriteHeader(http.StatusNotModified)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(testServer.Close)

	binDir := t.TempDir()
	oldGeoIP := mustGeoIPData(t, "OLD-IP")
	oldGeosite := mustGeositeData(t, "OLD-SITE")
	geoIPPath := filepath.Join(binDir, "geoip.dat")
	geositePath := filepath.Join(binDir, "geosite.dat")
	if err := os.WriteFile(geoIPPath, oldGeoIP, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(geositePath, oldGeosite, 0o600); err != nil {
		t.Fatal(err)
	}

	restartCalls := 0
	updater := geofileUpdater{
		client: testServer.Client(),
		binDir: binDir,
		entries: []geofileEntry{
			{URL: testServer.URL + "/geoip", FileName: "geoip.dat"},
			{URL: testServer.URL + "/geosite", FileName: "geosite.dat"},
		},
		restart: func() error {
			restartCalls++
			return nil
		},
	}

	if err := updater.update(""); err != nil {
		t.Fatal(err)
	}
	gotGeoIP, err := os.ReadFile(geoIPPath)
	if err != nil {
		t.Fatal(err)
	}
	gotGeosite, err := os.ReadFile(geositePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotGeoIP, newGeoIP) {
		t.Fatal("changed geoip was not committed")
	}
	if !bytes.Equal(gotGeosite, oldGeosite) {
		t.Fatal("304 geosite changed")
	}
	if restartCalls != 1 {
		t.Fatalf("restart calls = %d, want 1", restartCalls)
	}
	assertNoGeofileArtifacts(t, binDir)
}

func TestGeofileUpdateCommitFailureRollsBackAllFilesWithoutRestart(t *testing.T) {
	updater, paths, originals, restartCalls := newTwoFileCommitTestUpdater(t)
	replaceCalls := 0
	updater.replaceFile = func(source, destination string) error {
		replaceCalls++
		if replaceCalls == 2 {
			return errors.New("injected replace failure")
		}
		return os.Rename(source, destination)
	}

	if err := updater.update(""); err == nil {
		t.Fatal("expected commit failure")
	}
	for i, path := range paths {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, originals[i]) {
			t.Fatalf("file %s was not rolled back", filepath.Base(path))
		}
	}
	if *restartCalls != 0 {
		t.Fatalf("restart calls = %d, want 0", *restartCalls)
	}
	assertNoGeofileArtifacts(t, updater.binDir)
}

func TestGeofileUpdateXrayValidationFailureRollsBackWithoutRestart(t *testing.T) {
	updater, paths, originals, restartCalls := newTwoFileCommitTestUpdater(t)
	updater.validate = func() error { return errors.New("injected Xray validation failure") }

	if err := updater.update(""); err == nil {
		t.Fatal("expected Xray validation failure")
	}
	for i, path := range paths {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, originals[i]) {
			t.Fatalf("file %s was not rolled back", filepath.Base(path))
		}
	}
	if *restartCalls != 0 {
		t.Fatalf("restart calls = %d, want 0", *restartCalls)
	}
	assertNoGeofileArtifacts(t, updater.binDir)
}

func TestGeofileUpdateRestartFailureRollsBackAndRestartsPreviousConfig(t *testing.T) {
	updater, paths, originals, _ := newTwoFileCommitTestUpdater(t)
	updater.validate = func() error { return nil }
	restartCalls := 0
	updater.restart = func() error {
		restartCalls++
		if restartCalls == 1 {
			return errors.New("injected first restart failure")
		}
		return nil
	}

	if err := updater.update(""); err == nil {
		t.Fatal("expected update to report the first restart failure")
	}
	for i, path := range paths {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, originals[i]) {
			t.Fatalf("file %s was not restored after restart failure", filepath.Base(path))
		}
	}
	if restartCalls != 2 {
		t.Fatalf("restart calls = %d, want 2", restartCalls)
	}
	assertNoGeofileArtifacts(t, updater.binDir)
}

func TestGeofileUpdateRestartRollbackFailurePreservesRecoveryBackup(t *testing.T) {
	updater, _, _, _ := newTwoFileCommitTestUpdater(t)
	updater.validate = func() error { return nil }
	replaceCalls := 0
	updater.replaceFile = func(source, destination string) error {
		replaceCalls++
		if replaceCalls == 4 {
			return errors.New("injected rollback failure")
		}
		return os.Rename(source, destination)
	}
	restartCalls := 0
	updater.restart = func() error {
		restartCalls++
		return errors.New("injected first restart failure")
	}

	err := updater.update("")
	if err == nil {
		t.Fatal("expected rollback failure")
	}
	backups, globErr := filepath.Glob(filepath.Join(updater.binDir, ".geoip.dat-*.backup"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(backups) != 1 {
		t.Fatalf("recovery backups = %d, want 1", len(backups))
	}
	if !strings.Contains(err.Error(), backups[0]) {
		t.Fatalf("error does not identify preserved recovery backup: %v", err)
	}
	if restartCalls != 1 {
		t.Fatalf("restart calls = %d, want 1", restartCalls)
	}
}

func newTwoFileCommitTestUpdater(t *testing.T) (geofileUpdater, []string, [][]byte, *int) {
	t.Helper()
	newGeoIP := mustGeoIPData(t, "NEW-IP")
	newGeosite := mustGeositeData(t, "NEW-SITE")
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/geoip" {
			_, _ = w.Write(newGeoIP)
			return
		}
		_, _ = w.Write(newGeosite)
	}))
	t.Cleanup(testServer.Close)

	binDir := t.TempDir()
	originals := [][]byte{mustGeoIPData(t, "OLD-IP"), mustGeositeData(t, "OLD-SITE")}
	paths := []string{filepath.Join(binDir, "geoip.dat"), filepath.Join(binDir, "geosite.dat")}
	for i, path := range paths {
		if err := os.WriteFile(path, originals[i], 0o600); err != nil {
			t.Fatal(err)
		}
	}
	restartCalls := 0
	return geofileUpdater{
		client: testServer.Client(),
		binDir: binDir,
		entries: []geofileEntry{
			{URL: testServer.URL + "/geoip", FileName: "geoip.dat"},
			{URL: testServer.URL + "/geosite", FileName: "geosite.dat"},
		},
		restart: func() error {
			restartCalls++
			return nil
		},
	}, paths, originals, &restartCalls
}

func assertNoGeofileArtifacts(t *testing.T, binDir string) {
	t.Helper()
	artifacts, err := filepath.Glob(filepath.Join(binDir, ".*.dat-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 0 {
		t.Fatalf("unexpected geofile artifacts: %v", artifacts)
	}
}
