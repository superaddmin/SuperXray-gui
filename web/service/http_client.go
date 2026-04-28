package service

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxAPIResponseBytes      int64 = 4 * 1024 * 1024
	maxDownloadFileBytes     int64 = 128 * 1024 * 1024
	maxPublicIPResponseBytes int64 = 1024
	maxSubscriptionBytes     int64 = 4 * 1024 * 1024
)

var serviceHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

func readBodyLimited(body io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(body, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return data, nil
}

func copyLimited(dst io.Writer, src io.Reader, limit int64) error {
	written, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return err
	}
	if written > limit {
		return fmt.Errorf("response body exceeds %d bytes", limit)
	}
	return nil
}

func validateContentLength(resp *http.Response, limit int64) error {
	if resp.ContentLength > limit {
		return fmt.Errorf("response content length %d exceeds %d bytes", resp.ContentLength, limit)
	}
	return nil
}

func readSuccessfulBodyLimited(resp *http.Response, limit int64) ([]byte, error) {
	defer resp.Body.Close()
	if err := validateContentLength(resp, limit); err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, resp.Status)
	}
	return readBodyLimited(resp.Body, limit)
}
