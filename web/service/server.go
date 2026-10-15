package service

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/util/common"
	"github.com/superaddmin/SuperXray-gui/v2/util/pathutil"
	"github.com/superaddmin/SuperXray-gui/v2/util/sys"
	"github.com/superaddmin/SuperXray-gui/v2/xray"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// ProcessState represents the current state of a system process.
type ProcessState string

// Process state constants
const (
	Running             ProcessState = "running" // Process is running normally
	Stop                ProcessState = "stop"    // Process is stopped
	Error               ProcessState = "error"   // Process is in error state
	MaxImportDBFileSize int64        = 64 * 1024 * 1024
	xrayCommandTimeout               = 10 * time.Second
)

// Status represents comprehensive system and application status information.
// It includes CPU, memory, disk, network statistics, and Xray process status.
type Status struct {
	T           time.Time `json:"-"`
	Cpu         float64   `json:"cpu"`
	CpuCores    int       `json:"cpuCores"`
	LogicalPro  int       `json:"logicalPro"`
	CpuSpeedMhz float64   `json:"cpuSpeedMhz"`
	Mem         struct {
		Current uint64 `json:"current"`
		Total   uint64 `json:"total"`
	} `json:"mem"`
	Swap struct {
		Current uint64 `json:"current"`
		Total   uint64 `json:"total"`
	} `json:"swap"`
	Disk struct {
		Current uint64 `json:"current"`
		Total   uint64 `json:"total"`
	} `json:"disk"`
	Xray struct {
		State    ProcessState `json:"state"`
		ErrorMsg string       `json:"errorMsg"`
		Version  string       `json:"version"`
	} `json:"xray"`
	Uptime   uint64    `json:"uptime"`
	Loads    []float64 `json:"loads"`
	TcpCount int       `json:"tcpCount"`
	UdpCount int       `json:"udpCount"`
	NetIO    struct {
		Up   uint64 `json:"up"`
		Down uint64 `json:"down"`
	} `json:"netIO"`
	NetTraffic struct {
		Sent uint64 `json:"sent"`
		Recv uint64 `json:"recv"`
	} `json:"netTraffic"`
	PublicIP struct {
		IPv4 string `json:"ipv4"`
		IPv6 string `json:"ipv6"`
	} `json:"publicIP"`
	AppStats struct {
		Threads uint64 `json:"threads"`
		Mem     uint64 `json:"mem"`
		Uptime  uint64 `json:"uptime"`
	} `json:"appStats"`
}

// Release represents information about a software release from GitHub.
type Release struct {
	TagName string `json:"tag_name"` // The tag name of the release
}

// ServerService provides business logic for server monitoring and management.
// It handles system status collection, IP detection, and application statistics.
type ServerService struct {
	xrayService        XrayService
	cachedIPv4         string
	cachedIPv6         string
	noIPv6             bool
	mu                 sync.Mutex
	lastCPUTimes       cpu.TimesStat
	hasLastCPUSample   bool
	hasNativeCPUSample bool
	emaCPU             float64
	cpuHistory         []CPUSample
	cachedCpuSpeedMhz  float64
	lastCpuInfoAttempt time.Time
}

// AggregateCpuHistory returns up to maxPoints averaged buckets of size bucketSeconds over recent data.
func (s *ServerService) AggregateCpuHistory(bucketSeconds int, maxPoints int) []map[string]any {
	if bucketSeconds <= 0 || maxPoints <= 0 {
		return nil
	}
	cutoff := time.Now().Add(-time.Duration(bucketSeconds*maxPoints) * time.Second).Unix()
	s.mu.Lock()
	// find start index (history sorted ascending)
	hist := s.cpuHistory
	// binary-ish scan (simple linear from end since size capped ~10800 is fine)
	startIdx := 0
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].T < cutoff {
			startIdx = i + 1
			break
		}
	}
	if startIdx >= len(hist) {
		s.mu.Unlock()
		return []map[string]any{}
	}
	slice := hist[startIdx:]
	// copy for unlock
	tmp := make([]CPUSample, len(slice))
	copy(tmp, slice)
	s.mu.Unlock()
	if len(tmp) == 0 {
		return []map[string]any{}
	}
	var out []map[string]any
	var acc []float64
	bSize := int64(bucketSeconds)
	curBucket := (tmp[0].T / bSize) * bSize
	flush := func(ts int64) {
		if len(acc) == 0 {
			return
		}
		sum := 0.0
		for _, v := range acc {
			sum += v
		}
		avg := sum / float64(len(acc))
		out = append(out, map[string]any{"t": ts, "cpu": avg})
		acc = acc[:0]
	}
	for _, p := range tmp {
		b := (p.T / bSize) * bSize
		if b != curBucket {
			flush(curBucket)
			curBucket = b
		}
		acc = append(acc, p.Cpu)
	}
	flush(curBucket)
	if len(out) > maxPoints {
		out = out[len(out)-maxPoints:]
	}
	return out
}

// CPUSample single CPU utilization sample
type CPUSample struct {
	T   int64   `json:"t"`   // unix seconds
	Cpu float64 `json:"cpu"` // percent 0..100
}

type LogEntry struct {
	DateTime    time.Time
	FromAddress string
	ToAddress   string
	Inbound     string
	Outbound    string
	Email       string
	Event       int
}

func getPublicIP(url string) string {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "N/A"
	}
	defer resp.Body.Close()

	// Don't retry if access is blocked or region-restricted
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnavailableForLegalReasons {
		return "N/A"
	}
	if resp.StatusCode != http.StatusOK {
		return "N/A"
	}

	ip, err := readBodyLimited(resp.Body, maxPublicIPResponseBytes)
	if err != nil {
		return "N/A"
	}

	ipString := strings.TrimSpace(string(ip))
	if ipString == "" {
		return "N/A"
	}

	return ipString
}

func (s *ServerService) GetStatus(lastStatus *Status) *Status {
	now := time.Now()
	status := &Status{
		T: now,
	}

	// CPU stats
	util, err := s.sampleCPUUtilization()
	if err != nil {
		logger.Warning("get cpu percent failed:", err)
	} else {
		status.Cpu = util
	}

	status.CpuCores, err = cpu.Counts(false)
	if err != nil {
		logger.Warning("get cpu cores count failed:", err)
	}

	status.LogicalPro = runtime.NumCPU()

	if status.CpuSpeedMhz = s.cachedCpuSpeedMhz; s.cachedCpuSpeedMhz == 0 && time.Since(s.lastCpuInfoAttempt) > 5*time.Minute {
		s.lastCpuInfoAttempt = time.Now()
		done := make(chan struct{})
		go func() {
			defer close(done)
			cpuInfos, err := cpu.Info()
			if err != nil {
				logger.Warning("get cpu info failed:", err)
				return
			}
			if len(cpuInfos) > 0 {
				s.cachedCpuSpeedMhz = cpuInfos[0].Mhz
				status.CpuSpeedMhz = s.cachedCpuSpeedMhz
			} else {
				logger.Warning("could not find cpu info")
			}
		}()
		select {
		case <-done:
		case <-time.After(1500 * time.Millisecond):
			logger.Warning("cpu info query timed out; will retry later")
		}
	} else if s.cachedCpuSpeedMhz != 0 {
		status.CpuSpeedMhz = s.cachedCpuSpeedMhz
	}

	// Uptime
	upTime, err := host.Uptime()
	if err != nil {
		logger.Warning("get uptime failed:", err)
	} else {
		status.Uptime = upTime
	}

	// Memory stats
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		logger.Warning("get virtual memory failed:", err)
	} else {
		status.Mem.Current = memInfo.Used
		status.Mem.Total = memInfo.Total
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		logger.Warning("get swap memory failed:", err)
	} else {
		status.Swap.Current = swapInfo.Used
		status.Swap.Total = swapInfo.Total
	}

	// Disk stats
	diskInfo, err := disk.Usage("/")
	if err != nil {
		logger.Warning("get disk usage failed:", err)
	} else {
		status.Disk.Current = diskInfo.Used
		status.Disk.Total = diskInfo.Total
	}

	// Load averages
	avgState, err := load.Avg()
	if err != nil {
		logger.Warning("get load avg failed:", err)
	} else {
		status.Loads = []float64{avgState.Load1, avgState.Load5, avgState.Load15}
	}

	// Network stats
	ioStats, err := net.IOCounters(false)
	if err != nil {
		logger.Warning("get io counters failed:", err)
	} else if len(ioStats) > 0 {
		ioStat := ioStats[0]
		status.NetTraffic.Sent = ioStat.BytesSent
		status.NetTraffic.Recv = ioStat.BytesRecv

		if lastStatus != nil {
			duration := now.Sub(lastStatus.T)
			seconds := float64(duration) / float64(time.Second)
			up := uint64(float64(status.NetTraffic.Sent-lastStatus.NetTraffic.Sent) / seconds)
			down := uint64(float64(status.NetTraffic.Recv-lastStatus.NetTraffic.Recv) / seconds)
			status.NetIO.Up = up
			status.NetIO.Down = down
		}
	} else {
		logger.Warning("can not find io counters")
	}

	// TCP/UDP connections
	status.TcpCount, err = sys.GetTCPCount()
	if err != nil {
		logger.Warning("get tcp connections failed:", err)
	}

	status.UdpCount, err = sys.GetUDPCount()
	if err != nil {
		logger.Warning("get udp connections failed:", err)
	}

	// IP fetching with caching
	showIp4ServiceLists := []string{
		"https://api4.ipify.org",
		"https://ipv4.icanhazip.com",
		"https://v4.api.ipinfo.io/ip",
		"https://ipv4.myexternalip.com/raw",
		"https://4.ident.me",
		"https://check-host.net/ip",
	}
	showIp6ServiceLists := []string{
		"https://api6.ipify.org",
		"https://ipv6.icanhazip.com",
		"https://v6.api.ipinfo.io/ip",
		"https://ipv6.myexternalip.com/raw",
		"https://6.ident.me",
	}

	if s.cachedIPv4 == "" {
		for _, ip4Service := range showIp4ServiceLists {
			s.cachedIPv4 = getPublicIP(ip4Service)
			if s.cachedIPv4 != "N/A" {
				break
			}
		}
	}

	if s.cachedIPv6 == "" && !s.noIPv6 {
		for _, ip6Service := range showIp6ServiceLists {
			s.cachedIPv6 = getPublicIP(ip6Service)
			if s.cachedIPv6 != "N/A" {
				break
			}
		}
	}

	if s.cachedIPv6 == "N/A" {
		s.noIPv6 = true
	}

	status.PublicIP.IPv4 = s.cachedIPv4
	status.PublicIP.IPv6 = s.cachedIPv6

	// Xray status
	if s.xrayService.IsXrayRunning() {
		status.Xray.State = Running
		status.Xray.ErrorMsg = ""
	} else {
		err := s.xrayService.GetXrayErr()
		if err != nil {
			status.Xray.State = Error
		} else {
			status.Xray.State = Stop
		}
		status.Xray.ErrorMsg = s.xrayService.GetXrayResult()
	}
	status.Xray.Version = s.xrayService.GetXrayVersion()

	// Application stats
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	status.AppStats.Mem = rtm.Sys
	status.AppStats.Threads = goroutineCount()
	if p := currentXrayProcess(); p != nil && p.IsRunning() {
		status.AppStats.Uptime = p.GetUptime()
	} else {
		status.AppStats.Uptime = 0
	}

	return status
}

func goroutineCount() uint64 {
	count := runtime.NumGoroutine()
	value, err := strconv.ParseUint(strconv.Itoa(count), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func (s *ServerService) AppendCpuSample(t time.Time, v float64) {
	const capacity = 9000 // ~5 hours @ 2s interval
	s.mu.Lock()
	defer s.mu.Unlock()
	p := CPUSample{T: t.Unix(), Cpu: v}
	if n := len(s.cpuHistory); n > 0 && s.cpuHistory[n-1].T == p.T {
		s.cpuHistory[n-1] = p
	} else {
		s.cpuHistory = append(s.cpuHistory, p)
	}
	if len(s.cpuHistory) > capacity {
		s.cpuHistory = s.cpuHistory[len(s.cpuHistory)-capacity:]
	}
}

func (s *ServerService) sampleCPUUtilization() (float64, error) {
	// Try native platform-specific CPU implementation first (Windows, Linux, macOS)
	if pct, err := sys.CPUPercentRaw(); err == nil {
		s.mu.Lock()
		// First call to native method returns 0 (initializes baseline)
		if !s.hasNativeCPUSample {
			s.hasNativeCPUSample = true
			s.mu.Unlock()
			return 0, nil
		}
		// Smooth with EMA
		const alpha = 0.3
		if s.emaCPU == 0 {
			s.emaCPU = pct
		} else {
			s.emaCPU = alpha*pct + (1-alpha)*s.emaCPU
		}
		val := s.emaCPU
		s.mu.Unlock()
		return val, nil
	}
	// If native call fails, fall back to gopsutil times
	// Read aggregate CPU times (all CPUs combined)
	times, err := cpu.Times(false)
	if err != nil {
		return 0, err
	}
	if len(times) == 0 {
		return 0, fmt.Errorf("no cpu times available")
	}

	cur := times[0]

	s.mu.Lock()
	defer s.mu.Unlock()

	// If this is the first sample, initialize and return current EMA (0 by default)
	if !s.hasLastCPUSample {
		s.lastCPUTimes = cur
		s.hasLastCPUSample = true
		return s.emaCPU, nil
	}

	// Compute busy and total deltas
	// Note: Guest and GuestNice times are already included in User and Nice respectively,
	// so we exclude them to avoid double-counting (Linux kernel accounting)
	idleDelta := cur.Idle - s.lastCPUTimes.Idle
	busyDelta := (cur.User - s.lastCPUTimes.User) +
		(cur.System - s.lastCPUTimes.System) +
		(cur.Nice - s.lastCPUTimes.Nice) +
		(cur.Iowait - s.lastCPUTimes.Iowait) +
		(cur.Irq - s.lastCPUTimes.Irq) +
		(cur.Softirq - s.lastCPUTimes.Softirq) +
		(cur.Steal - s.lastCPUTimes.Steal)

	totalDelta := busyDelta + idleDelta

	// Update last sample for next time
	s.lastCPUTimes = cur

	// Guard against division by zero or negative deltas (e.g., counter resets)
	if totalDelta <= 0 {
		return s.emaCPU, nil
	}

	raw := 100.0 * (busyDelta / totalDelta)
	if raw < 0 {
		raw = 0
	}
	if raw > 100 {
		raw = 100
	}

	// Exponential moving average to smooth spikes
	const alpha = 0.3 // smoothing factor (0<alpha<=1). Higher = more responsive, lower = smoother
	if s.emaCPU == 0 {
		// Initialize EMA with the first real reading to avoid long warm-up from zero
		s.emaCPU = raw
	} else {
		s.emaCPU = alpha*raw + (1-alpha)*s.emaCPU
	}

	return s.emaCPU, nil
}

func (s *ServerService) GetXrayVersions() ([]string, error) {
	const (
		XrayURL = "https://api.github.com/repos/XTLS/Xray-core/releases"
	)

	req, err := http.NewRequest(http.MethodGet, XrayURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := validateContentLength(resp, maxAPIResponseBytes); err != nil {
		return nil, err
	}

	// Check HTTP status code - GitHub API returns object instead of array on error
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := readBodyLimited(resp.Body, maxAPIResponseBytes)
		var errorResponse struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(bodyBytes, &errorResponse) == nil && errorResponse.Message != "" {
			return nil, fmt.Errorf("GitHub API error: %s", errorResponse.Message)
		}
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	bodyBytes, err := readBodyLimited(resp.Body, maxAPIResponseBytes)
	if err != nil {
		return nil, err
	}

	var releases []Release
	if err := json.Unmarshal(bodyBytes, &releases); err != nil {
		return nil, err
	}

	var versions []string
	for _, release := range releases {
		if supportedCoreVersion(release.TagName) {
			versions = append(versions, release.TagName)
		}
	}
	return versions, nil
}

func (s *ServerService) StopXrayService() error {
	err := s.xrayService.StopXray()
	if err != nil {
		logger.Error("stop xray failed:", err)
		return err
	}
	return nil
}

func (s *ServerService) RestartXrayService() error {
	err := s.xrayService.RestartXray(true)
	if err != nil {
		logger.Error("start xray failed:", err)
		return err
	}
	return nil
}

func (s *ServerService) downloadXRay(version string) (string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	switch osName {
	case "darwin":
		osName = "macos"
	case "windows":
		osName = "windows"
	}

	switch arch {
	case "amd64":
		arch = "64"
	case "arm64":
		arch = "arm64-v8a"
	case "armv7":
		arch = "arm32-v7a"
	case "armv6":
		arch = "arm32-v6"
	case "armv5":
		arch = "arm32-v5"
	case "386":
		arch = "32"
	case "s390x":
		arch = "s390x"
	}

	fileName := fmt.Sprintf("Xray-%s-%s.zip", osName, arch)
	url := fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/%s", version, fileName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download Xray failed with status %d: %s", resp.StatusCode, resp.Status)
	}
	if err := validateContentLength(resp, maxDownloadFileBytes); err != nil {
		return "", err
	}

	if err := os.MkdirAll(config.GetBinFolderPath(), 0o750); err != nil {
		return "", err
	}
	downloadDir, err := os.MkdirTemp(config.GetBinFolderPath(), ".xray-download-*")
	if err != nil {
		return "", err
	}
	downloadPath := filepath.Join(downloadDir, fileName)
	// #nosec G304 -- fixed release asset basename inside a fresh private download directory.
	file, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		_ = os.RemoveAll(downloadDir)
		return "", err
	}
	copyErr := copyLimited(file, resp.Body, maxDownloadFileBytes)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		_ = os.RemoveAll(downloadDir)
		return "", err
	}
	return downloadPath, nil
}

func (s *ServerService) verifyXrayArchive(path string, version string) error {
	// #nosec G304 -- caller supplies only the private archive returned by downloadXRay.
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxDownloadFileBytes+1))
	if err != nil {
		return err
	}
	if n > maxDownloadFileBytes {
		return fmt.Errorf("xray archive exceeds %d bytes", maxDownloadFileBytes)
	}
	actual := hex.EncodeToString(h.Sum(nil))
	name := filepath.Base(path)
	url := fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/%s.dgst", version, name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("xray digest download failed with status %d", resp.StatusCode)
	}
	body, err := readBodyLimited(resp.Body, 64*1024)
	if err != nil {
		return err
	}
	var expected string
	for _, line := range strings.Split(string(body), "\n") {
		field := strings.Fields(line)
		if len(field) == 2 && field[0] == "SHA2-256=" && len(field[1]) == 64 {
			if _, e := hex.DecodeString(field[1]); e == nil && expected == "" {
				expected = strings.ToLower(field[1])
			} else {
				return fmt.Errorf("invalid or duplicate SHA-256 digest for %s", name)
			}
		}
	}
	if expected == "" {
		return fmt.Errorf("xray digest file contains no SHA-256 for %s", name)
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("xray archive SHA-256 mismatch: got %s, want %s", actual, expected)
	}
	return nil
}

func (s *ServerService) UpdateXray(version string) error {
	runtimeMaintenanceMu.Lock()
	defer runtimeMaintenanceMu.Unlock()
	version = strings.TrimSpace(version)
	if !supportedCoreVersion(version) {
		return fmt.Errorf("xray version must be a stable vYY.M.D release at least v26.10.10")
	}
	// Download and authenticate the archive before interrupting a running core.
	zipFileName, err := s.downloadXRay(version)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(xray.GetBinaryPath()), 0o750); err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(zipFileName))
	if err := s.verifyXrayArchive(zipFileName, version); err != nil {
		return err
	}

	// #nosec G304 -- SHA-256 verified archive in this operation's private download directory.
	zipFile, err := os.Open(zipFileName)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	stat, err := zipFile.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(zipFile, stat.Size())
	if err != nil {
		return err
	}

	stageDir, err := os.MkdirTemp(filepath.Dir(xray.GetBinaryPath()), ".xray-update-*")
	if err != nil {
		return err
	}
	cleanupStage := true
	defer func() {
		if cleanupStage {
			_ = os.RemoveAll(stageDir)
		}
	}()
	// Extract only the expected files into a private staging directory.
	copyZipFile := func(zipName string, fileName string) error {
		zipFile, err := reader.Open(zipName)
		if err != nil {
			return err
		}
		defer zipFile.Close()
		if err := os.MkdirAll(filepath.Dir(fileName), 0o750); err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if strings.HasSuffix(fileName, "xray") || strings.HasSuffix(fileName, "xray.exe") {
			mode = 0o750
		}
		// #nosec G304 -- destination is one of three fixed filenames in the private staging directory.
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return err
		}
		copyErr := copyLimited(file, zipFile, maxDownloadFileBytes)
		return errors.Join(copyErr, file.Sync(), file.Close())
	}

	// Extract correct binary and data files.
	stageBinary := filepath.Join(stageDir, "xray")
	if runtime.GOOS == "windows" {
		stageBinary += ".exe"
	}
	stageGeoip := filepath.Join(stageDir, "geoip.dat")
	stageGeosite := filepath.Join(stageDir, "geosite.dat")
	if runtime.GOOS == "windows" {
		err = copyZipFile("xray.exe", stageBinary)
	} else {
		err = copyZipFile("xray", stageBinary)
	}
	if err != nil {
		return err
	}
	if err = copyZipFile("geoip.dat", stageGeoip); err != nil {
		return err
	}
	if err = copyZipFile("geosite.dat", stageGeosite); err != nil {
		return err
	}
	candidateConfig, err := s.xrayService.getXrayConfig(false)
	if err != nil {
		return err
	}
	assetDir, err := coreAssetDirectory(candidateConfig)
	if err != nil {
		return err
	}
	if err := stageSupplementalGeo(assetDir, stageDir); err != nil {
		return err
	}
	if err := xray.ValidateConfigWithBinary(candidateConfig, stageBinary); err != nil {
		return fmt.Errorf("candidate Xray cannot use the current configuration: %w", err)
	}
	files := []coreFileReplacement{
		{target: xray.GetBinaryPath(), staged: stageBinary, backup: filepath.Join(stageDir, "old-xray")},
		{target: xray.GetGeoipPath(), staged: stageGeoip, backup: filepath.Join(stageDir, "old-geoip.dat")},
		{target: xray.GetGeositePath(), staged: stageGeosite, backup: filepath.Join(stageDir, "old-geosite.dat")},
	}
	retain, err := installCoreFiles(files, coreUpdateLifecycle{
		running: s.xrayService.IsXrayRunning,
		stop:    s.xrayService.stopXray,
		start:   func() error { return s.xrayService.restartXray(true) },
	})
	cleanupStage = !retain
	if retain {
		return fmt.Errorf("xray update recovery incomplete; backups retained in %s: %w", stageDir, err)
	}
	return err
}

func (s *ServerService) GetLogs(count string, level string, syslog string) []string {
	c, _ := strconv.Atoi(count)
	var lines []string

	if syslog == "true" {
		// Check if running on Windows - journalctl is not available
		if runtime.GOOS == "windows" {
			return []string{"Syslog is not supported on Windows. Please use application logs instead by unchecking the 'Syslog' option."}
		}

		// Validate and sanitize count parameter
		countInt, err := strconv.Atoi(count)
		if err != nil || countInt < 1 || countInt > 10000 {
			return []string{"Invalid count parameter - must be a number between 1 and 10000"}
		}

		// Validate level parameter - only allow valid syslog levels
		validLevels := map[string]bool{
			"0": true, "emerg": true,
			"1": true, "alert": true,
			"2": true, "crit": true,
			"3": true, "err": true,
			"4": true, "warning": true,
			"5": true, "notice": true,
			"6": true, "info": true,
			"7": true, "debug": true,
		}
		if !validLevels[level] {
			return []string{"Invalid level parameter - must be a valid syslog level"}
		}

		// Use hardcoded command with validated parameters
		ctx, cancel := context.WithTimeout(context.Background(), xrayCommandTimeout)
		defer cancel()
		// #nosec G204 -- command is fixed and all variable arguments are allowlisted above.
		cmd := exec.CommandContext(ctx, "journalctl", "-u", "x-ui", "--no-pager", "-n", strconv.Itoa(countInt), "-p", level)
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return []string{"journalctl command timed out."}
			}
			return []string{"Failed to run journalctl command! Make sure systemd is available and x-ui service is registered."}
		}
		lines = strings.Split(out.String(), "\n")
	} else {
		lines = logger.GetLogs(c, level)
	}

	return lines
}

func (s *ServerService) GetXrayLogs(
	count string,
	filter string,
	showDirect string,
	showBlocked string,
	showProxy string,
	freedoms []string,
	blackholes []string,
) []LogEntry {
	const (
		Direct = iota
		Blocked
		Proxied
	)

	countInt, err := strconv.Atoi(count)
	if err != nil || countInt < 1 || countInt > 10000 {
		return nil
	}
	var entries []LogEntry

	pathToAccessLog, err := xray.GetAccessLogPath()
	if err != nil {
		return nil
	}

	file, err := pathutil.OpenUnder(filepath.Dir(pathToAccessLog), pathToAccessLog)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.Contains(line, "api -> api") {
			// skipping empty lines and api calls
			continue
		}

		if filter != "" && !strings.Contains(line, filter) {
			// applying filter if it's not empty
			continue
		}

		var entry LogEntry
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		dateTime, err := time.ParseInLocation("2006/01/02 15:04:05.999999", parts[0]+" "+parts[1], time.Local)
		if err != nil {
			continue
		}
		entry.DateTime = dateTime.UTC()

		for i, part := range parts {
			if (part == "from" || part == "accepted" || part == "email:") && i+1 >= len(parts) {
				continue
			}

			if part == "from" {
				entry.FromAddress = strings.TrimLeft(parts[i+1], "/")
			} else if part == "accepted" {
				entry.ToAddress = strings.TrimLeft(parts[i+1], "/")
			} else if strings.HasPrefix(part, "[") {
				entry.Inbound = part[1:]
			} else if strings.HasSuffix(part, "]") {
				entry.Outbound = part[:len(part)-1]
			} else if part == "email:" {
				entry.Email = parts[i+1]
			}
		}

		if logEntryContains(line, freedoms) {
			if showDirect == "false" {
				continue
			}
			entry.Event = Direct
		} else if logEntryContains(line, blackholes) {
			if showBlocked == "false" {
				continue
			}
			entry.Event = Blocked
		} else {
			if showProxy == "false" {
				continue
			}
			entry.Event = Proxied
		}

		entries = append(entries, entry)
	}

	if len(entries) > countInt {
		entries = entries[len(entries)-countInt:]
	}

	return entries
}

func logEntryContains(line string, suffixes []string) bool {
	for _, sfx := range suffixes {
		if strings.Contains(line, sfx+"]") {
			return true
		}
	}
	return false
}

func (s *ServerService) GetConfigJson() (any, error) {
	config, err := s.xrayService.GetXrayConfig()
	if err != nil {
		return nil, err
	}
	contents, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}

	var jsonData any
	err = json.Unmarshal(contents, &jsonData)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (s *ServerService) GetDb() ([]byte, error) {
	file, err := os.CreateTemp(config.GetDBFolderPath(), ".superxray-snapshot-*.db")
	if err != nil {
		return nil, err
	}
	snapshot := file.Name()
	defer os.Remove(snapshot)
	if err := file.Close(); err != nil {
		return nil, err
	}
	if err := os.Remove(snapshot); err != nil {
		return nil, err
	}
	if err := database.BackupSQLite(snapshot); err != nil {
		return nil, err
	}
	// #nosec G304 -- snapshot was allocated by CreateTemp in the configured DB directory.
	return os.ReadFile(snapshot)
}

func (s *ServerService) ImportDB(file multipart.File) error {
	releaseDatabase := database.AcquireMaintenance()
	defer releaseDatabase()
	runtimeMaintenanceMu.Lock()
	defer runtimeMaintenanceMu.Unlock()
	// Check if the file is a SQLite database
	isValidDb, err := database.IsSQLiteDB(file)
	if err != nil {
		return common.NewErrorf("Error checking db file format: %v", err)
	}
	if !isValidDb {
		return common.NewError("Invalid db file format")
	}

	// Reset the file reader to the beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		return common.NewErrorf("Error resetting file reader: %v", err)
	}

	// Each import owns a private candidate; never reuse a stale upload path.
	tempFile, err := os.CreateTemp(config.GetDBFolderPath(), ".superxray-import-*.db")
	if err != nil {
		return common.NewErrorf("Error creating temporary db file: %v", err)
	}
	tempPath := tempFile.Name()

	// Robust deferred cleanup for the temporary file
	defer func() {
		if tempFile != nil {
			if cerr := tempFile.Close(); cerr != nil {
				logger.Warningf("Warning: failed to close temp file: %v", cerr)
			}
		}
		if _, err := os.Stat(tempPath); err == nil {
			if rerr := os.Remove(tempPath); rerr != nil {
				logger.Warningf("Warning: failed to remove temp file: %v", rerr)
			}
		}
		for _, suffix := range []string{"-wal", "-shm", "-journal"} {
			if rerr := os.Remove(tempPath + suffix); rerr != nil && !os.IsNotExist(rerr) {
				logger.Warningf("Warning: failed to remove temporary database sidecar: %v", rerr)
			}
		}
	}()

	// Save uploaded file to temporary file
	if err = copyLimited(tempFile, file, MaxImportDBFileSize); err != nil {
		return common.NewErrorf("Error saving db: %v", err)
	}
	if err = tempFile.Sync(); err != nil {
		return common.NewErrorf("Error syncing temporary db file: %v", err)
	}

	// Close temp file before opening via sqlite
	if err = tempFile.Close(); err != nil {
		return common.NewErrorf("Error closing temporary db file: %v", err)
	}
	tempFile = nil

	// Validate integrity (no migrations / side effects)
	if err = database.ValidateSQLiteDB(tempPath); err != nil {
		return common.NewErrorf("Invalid or corrupt db file: %v", err)
	}

	// Migrate and validate the complete candidate while the original database
	// and core remain available. Candidate repositories never publish globally.
	if err := validateImportCandidate(tempPath); err != nil {
		return common.NewErrorf("Imported database is incompatible: %v", err)
	}
	return replacePreparedDatabase(tempPath, config.GetDBPath(), databaseImportLifecycle{
		running:    s.xrayService.IsXrayRunning(),
		stop:       s.xrayService.stopXray,
		start:      func() error { return s.xrayService.restartXray(true) },
		checkpoint: database.Checkpoint,
		close:      database.CloseDB,
		open:       InitDB,
	})
}

func runXrayCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), xrayCommandTimeout)
	defer cancel()

	// #nosec G204 -- Xray binary path is project-controlled and arguments are passed without a shell.
	cmd := exec.CommandContext(ctx, xray.GetBinaryPath(), args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("xray command timed out after %s", xrayCommandTimeout)
		}
		return "", err
	}
	return out.String(), nil
}

func parseCommandValue(lines []string, index int, field string) (string, error) {
	if len(lines) <= index {
		return "", common.NewErrorf("invalid xray output: missing %s", field)
	}
	parts := strings.SplitN(lines[index], ":", 2)
	if len(parts) != 2 {
		return "", common.NewErrorf("invalid xray output: missing separator for %s", field)
	}
	value := strings.TrimSpace(parts[1])
	if value == "" {
		return "", common.NewErrorf("invalid xray output: empty %s", field)
	}
	return value, nil
}

func validateECHServerName(sni string) error {
	sni = strings.TrimSpace(sni)
	if sni == "" {
		return common.NewError("server name is required")
	}
	if strings.HasPrefix(sni, "-") || strings.ContainsAny(sni, " \t\r\n\x00") {
		return common.NewError("invalid server name")
	}
	return nil
}

func (s *ServerService) GetNewX25519Cert() (any, error) {
	output, err := runXrayCommand("x25519")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")

	privateKey, err := parseCommandValue(lines, 0, "private key")
	if err != nil {
		return nil, err
	}
	publicKey, err := parseCommandValue(lines, 1, "public key")
	if err != nil {
		return nil, err
	}

	keyPair := map[string]any{
		"privateKey": privateKey,
		"publicKey":  publicKey,
	}

	return keyPair, nil
}

func (s *ServerService) GetNewmldsa65() (any, error) {
	output, err := runXrayCommand("mldsa65")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")

	seed, err := parseCommandValue(lines, 0, "seed")
	if err != nil {
		return nil, err
	}
	verify, err := parseCommandValue(lines, 1, "verify")
	if err != nil {
		return nil, err
	}

	keyPair := map[string]any{
		"seed":   seed,
		"verify": verify,
	}

	return keyPair, nil
}

func (s *ServerService) GetNewEchCert(sni string) (any, error) {
	if err := validateECHServerName(sni); err != nil {
		return nil, err
	}

	output, err := runXrayCommand("tls", "ech", "--serverName", strings.TrimSpace(sni))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	if len(lines) < 4 {
		return nil, common.NewError("invalid ech cert")
	}

	configList := lines[1]
	serverKeys := lines[3]

	return map[string]any{
		"echServerKeys": serverKeys,
		"echConfigList": configList,
	}, nil
}

func (s *ServerService) GetNewVlessEnc() (any, error) {
	output, err := runXrayCommand("vlessenc")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	var auths []map[string]string
	var current map[string]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Authentication:") {
			if current != nil {
				auths = append(auths, current)
			}
			current = map[string]string{
				"label": strings.TrimSpace(strings.TrimPrefix(line, "Authentication:")),
			}
		} else if strings.HasPrefix(line, `"decryption"`) || strings.HasPrefix(line, `"encryption"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && current != nil {
				key := strings.Trim(parts[0], `" `)
				val := strings.Trim(parts[1], `" `)
				current[key] = val
			}
		}
	}

	if current != nil {
		auths = append(auths, current)
	}

	return map[string]any{
		"auths": auths,
	}, nil
}

func (s *ServerService) GetNewUUID() (map[string]string, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	return map[string]string{
		"uuid": newUUID.String(),
	}, nil
}

func (s *ServerService) GetNewmlkem768() (any, error) {
	output, err := runXrayCommand("mlkem768")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")

	seed, err := parseCommandValue(lines, 0, "seed")
	if err != nil {
		return nil, err
	}
	client, err := parseCommandValue(lines, 1, "client")
	if err != nil {
		return nil, err
	}

	keyPair := map[string]any{
		"seed":   seed,
		"client": client,
	}

	return keyPair, nil
}
