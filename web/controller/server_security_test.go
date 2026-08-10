package controller

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateImportDBFileSizeRejectsOversizedUpload(t *testing.T) {
	if err := validateImportDBFileSize(maxImportDBFileSize + 1); err == nil {
		t.Fatal("expected oversized upload to be rejected")
	}
}

func TestValidateImportDBFileSizeAllowsConfiguredLimit(t *testing.T) {
	if err := validateImportDBFileSize(maxImportDBFileSize); err != nil {
		t.Fatalf("expected upload at configured limit to be allowed: %v", err)
	}
}

func TestValidateImportDBFileSizeRejectsTinyUpload(t *testing.T) {
	if err := validateImportDBFileSize(minImportDBFileSize - 1); err == nil {
		t.Fatal("expected tiny upload to be rejected")
	}
}

func TestValidateImportDBUploadMetadataAllowsSQLiteNames(t *testing.T) {
	for _, filename := range []string{"x-ui.db", "backup.sqlite", "backup.sqlite3"} {
		if err := validateImportDBUploadMetadata(filename, minImportDBFileSize); err != nil {
			t.Fatalf("expected %q to be allowed: %v", filename, err)
		}
	}
}

func TestValidateImportDBUploadMetadataRejectsUnsafeNames(t *testing.T) {
	for _, filename := range []string{"../x-ui.db", `nested\x-ui.db`, "x-ui.txt", "x-ui.db.exe", ""} {
		if err := validateImportDBUploadMetadata(filename, minImportDBFileSize); err == nil {
			t.Fatalf("expected %q to be rejected", filename)
		}
	}
}

func TestServerRoutesExposeSelfSignedCertificateAsPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller := &ServerController{}
	controller.initRouter(router.Group("/panel/api/server"))

	for _, route := range router.Routes() {
		if route.Method == "POST" && route.Path == "/panel/api/server/getNewSelfSignedCert" {
			return
		}
	}
	t.Fatal("self-signed certificate POST route is missing")
}

func TestSelfSignedCertificateEndpointRejectsOversizedFormBodies(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        func(t *testing.T) []byte
	}{
		{
			name:        "url encoded",
			contentType: "application/x-www-form-urlencoded",
			body: func(t *testing.T) []byte {
				t.Helper()
				values := url.Values{"sni": {strings.Repeat("a", int(maxSelfSignedCertRequestSize))}}
				return []byte(values.Encode())
			},
		},
		{
			name: "multipart",
			body: func(t *testing.T) []byte {
				t.Helper()
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				if err := writer.WriteField("sni", strings.Repeat("a", int(maxSelfSignedCertRequestSize))); err != nil {
					t.Fatalf("write multipart field: %v", err)
				}
				if err := writer.Close(); err != nil {
					t.Fatalf("close multipart body: %v", err)
				}
				return append([]byte(writer.FormDataContentType()+"\n"), body.Bytes()...)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := test.body(t)
			contentType := test.contentType
			if test.name == "multipart" {
				parts := bytes.SplitN(body, []byte("\n"), 2)
				contentType = string(parts[0])
				body = parts[1]
			}

			gin.SetMode(gin.TestMode)
			router := gin.New()
			controller := &ServerController{}
			controller.initRouter(router.Group("/panel/api/server"))

			request := httptest.NewRequest(
				http.MethodPost,
				"/panel/api/server/getNewSelfSignedCert",
				bytes.NewReader(body),
			)
			request.Header.Set("Content-Type", contentType)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
			}
			if response.Header().Get("Pragma") != "no-cache" {
				t.Fatalf("Pragma = %q", response.Header().Get("Pragma"))
			}

			var result struct {
				Success bool `json:"success"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if result.Success {
				t.Fatal("oversized certificate request was accepted")
			}
		})
	}
}
