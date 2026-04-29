package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/util/common"
)

// WarpService provides business logic for Cloudflare WARP integration.
// It manages WARP configuration and connectivity settings.
type WarpService struct {
	SettingService
}

type warpRegistrationPayload struct {
	Key   string `json:"key"`
	TOS   string `json:"tos"`
	Type  string `json:"type"`
	Model string `json:"model"`
	Name  string `json:"name"`
}

type warpLicensePayload struct {
	License string `json:"license"`
}

type warpStoredData struct {
	AccessToken string `json:"access_token"`
	DeviceID    string `json:"device_id"`
	LicenseKey  string `json:"license_key"`
	PrivateKey  string `json:"private_key"`
}

func buildWarpRegistrationPayload(publicKey, tos, hostName string) ([]byte, error) {
	return json.Marshal(warpRegistrationPayload{
		Key:   publicKey,
		TOS:   tos,
		Type:  "PC",
		Model: "x-ui",
		Name:  hostName,
	})
}

func buildWarpLicensePayload(license string) ([]byte, error) {
	return json.Marshal(warpLicensePayload{License: license})
}

func buildWarpStoredData(token, deviceID, license, secretKey string) ([]byte, error) {
	// #nosec G117 -- WARP export data intentionally stores provider-issued credentials for Xray config use.
	return json.MarshalIndent(warpStoredData{
		AccessToken: token,
		DeviceID:    deviceID,
		LicenseKey:  license,
		PrivateKey:  secretKey,
	}, "", "  ")
}

func buildWarpResult(storedData []byte, config []byte) ([]byte, error) {
	result := map[string]json.RawMessage{
		"data":   json.RawMessage(storedData),
		"config": json.RawMessage(config),
	}
	return json.MarshalIndent(result, "", "  ")
}

func (s *WarpService) GetWarpData() (string, error) {
	warp, err := s.SettingService.GetWarp()
	if err != nil {
		return "", err
	}
	return warp, nil
}

func (s *WarpService) DelWarpData() error {
	err := s.SettingService.SetWarp("")
	if err != nil {
		return err
	}
	return nil
}

func (s *WarpService) GetWarpConfig() (string, error) {
	var warpData map[string]string
	warp, err := s.SettingService.GetWarp()
	if err != nil {
		return "", err
	}
	err = json.Unmarshal([]byte(warp), &warpData)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.cloudflareclient.com/v0a2158/reg/%s", warpData["device_id"])

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+warpData["access_token"])

	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := validateContentLength(resp, maxAPIResponseBytes); err != nil {
		return "", err
	}
	body, err := readBodyLimited(resp.Body, maxAPIResponseBytes)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *WarpService) RegWarp(secretKey string, publicKey string) (string, error) {
	tos := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	hostName, _ := os.Hostname()
	data, err := buildWarpRegistrationPayload(publicKey, tos, hostName)
	if err != nil {
		return "", err
	}

	url := "https://api.cloudflareclient.com/v0a2158/reg"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Add("CF-Client-Version", "a-7.21-0721")
	req.Header.Add("Content-Type", "application/json")

	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := validateContentLength(resp, maxAPIResponseBytes); err != nil {
		return "", err
	}
	body, err := readBodyLimited(resp.Body, maxAPIResponseBytes)
	if err != nil {
		return "", err
	}

	var rspData map[string]any
	err = json.Unmarshal(body, &rspData)
	if err != nil {
		return "", err
	}

	deviceId, ok := rspData["id"].(string)
	if !ok || deviceId == "" {
		return "", common.NewError("missing WARP device id")
	}
	token, ok := rspData["token"].(string)
	if !ok || token == "" {
		return "", common.NewError("missing WARP access token")
	}
	account, ok := rspData["account"].(map[string]any)
	if !ok {
		return "", common.NewError("missing WARP account data")
	}
	license, ok := account["license"].(string)
	if !ok {
		logger.Debug("Error accessing license value.")
		return "", common.NewError("missing WARP license")
	}

	warpData, err := buildWarpStoredData(token, deviceId, license, secretKey)
	if err != nil {
		return "", err
	}

	if err := s.SettingService.SetWarp(string(warpData)); err != nil {
		return "", err
	}

	result, err := buildWarpResult(warpData, body)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (s *WarpService) SetWarpLicense(license string) (string, error) {
	var warpData map[string]string
	warp, err := s.SettingService.GetWarp()
	if err != nil {
		return "", err
	}
	err = json.Unmarshal([]byte(warp), &warpData)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.cloudflareclient.com/v0a2158/reg/%s/account", warpData["device_id"])
	data, err := buildWarpLicensePayload(license)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+warpData["access_token"])

	resp, err := serviceHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := validateContentLength(resp, maxAPIResponseBytes); err != nil {
		return "", err
	}
	body, err := readBodyLimited(resp.Body, maxAPIResponseBytes)
	if err != nil {
		return "", err
	}

	var response map[string]any
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}
	if response["success"] == false {
		errorArr, _ := response["errors"].([]any)
		errorObj := errorArr[0].(map[string]any)
		return "", common.NewError(errorObj["code"], errorObj["message"])
	}

	warpData["license_key"] = license
	newWarpData, err := json.MarshalIndent(warpData, "", "  ")
	if err != nil {
		return "", err
	}
	if err := s.SettingService.SetWarp(string(newWarpData)); err != nil {
		return "", err
	}

	return string(newWarpData), nil
}
