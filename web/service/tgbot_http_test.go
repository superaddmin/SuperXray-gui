package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/valyala/fasthttp"
)

type telegramRoundTripFunc func(*fasthttp.HostClient, *fasthttp.Request, *fasthttp.Response) (bool, error)

func (f telegramRoundTripFunc) RoundTrip(client *fasthttp.HostClient, request *fasthttp.Request, response *fasthttp.Response) (bool, error) {
	return f(client, request, response)
}

func TestTgbotHTTPClientRetriesOnlyGetAndPost(t *testing.T) {
	transportError := errors.New("injected connection failure")
	for _, test := range []struct {
		name         string
		method       string
		failures     int
		wantAttempts int
		wantError    bool
	}{
		{"get recovers", http.MethodGet, 1, 2, false},
		{"post recovers", http.MethodPost, 1, 2, false},
		{"get retry limit", http.MethodGet, 3, 3, true},
		{"post retry limit", http.MethodPost, 3, 3, true},
		{"head does not retry", http.MethodHead, 1, 1, true},
		{"put does not retry", http.MethodPut, 1, 1, true},
		{"delete does not retry", http.MethodDelete, 1, 1, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := (&Tgbot{}).createRobustFastHTTPClient("")
			attempts := 0
			client.Transport = telegramRoundTripFunc(func(_ *fasthttp.HostClient, _ *fasthttp.Request, response *fasthttp.Response) (bool, error) {
				attempts++
				if attempts <= test.failures {
					return true, transportError
				}
				response.SetStatusCode(http.StatusOK)
				return false, nil
			})
			defer client.CloseIdleConnections()

			request := fasthttp.AcquireRequest()
			defer fasthttp.ReleaseRequest(request)
			request.SetRequestURI("http://telegram.example.test/test")
			request.Header.SetMethod(test.method)
			response := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(response)

			if client.RetryIfErr == nil {
				t.Fatal("error-aware retry callback is not configured")
			}
			if resetTimeout, _ := client.RetryIfErr(request, 1, transportError); resetTimeout {
				t.Fatal("retry must preserve the original request deadline")
			}
			err := client.Do(request, response)
			if test.wantError && !errors.Is(err, transportError) {
				t.Fatalf("Do error = %v, want injected transport error", err)
			}
			if !test.wantError && err != nil {
				t.Fatalf("Do returned error: %v", err)
			}
			if attempts != test.wantAttempts {
				t.Fatalf("attempts = %d, want %d", attempts, test.wantAttempts)
			}
		})
	}
}
