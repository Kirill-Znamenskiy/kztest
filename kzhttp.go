package kztest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type HTTPTestKitRequest struct {
	Method          string
	Target          string
	Headers         map[string]string
	Body            string
	BodyMakeFunc    func(t *testing.T, tkit *HTTPTestKit) string
	HeadersMakeFunc func(t *testing.T, tkit *HTTPTestKit) map[string]string
}

type HTTPTestKitResponse struct {
	StatusCode    int
	Headers       map[string]string
	Body          string
	BodyCheckFunc func(t *testing.T, respBody string, tkit *HTTPTestKit) bool
}
type HTTPTestKit struct {
	Request                  HTTPTestKitRequest
	Response                 HTTPTestKitResponse
	BeforePerformRequestFunc func(t *testing.T, req *http.Request, tkit *HTTPTestKit)
	AfterPerformRequestFunc  func(t *testing.T, req *http.Request, resp *http.Response, tkit *HTTPTestKit)
}

func RunHTTPTests(t *testing.T, httpHandler http.Handler, tkits []HTTPTestKit) {

	for tind, tkit := range tkits {
		t.Run(fmt.Sprintf("(%d) Test %s %s", tind+1, tkit.Request.Method, tkit.Request.Target), func(t *testing.T) {

			w := httptest.NewRecorder()

			var tkitReqBody io.Reader
			if tkit.Request.Body != "" {
				tkitReqBody = strings.NewReader(tkit.Request.Body)
			} else if tkit.Request.BodyMakeFunc != nil {
				tkitReqBody = strings.NewReader(tkit.Request.BodyMakeFunc(t, &tkit))
			}
			req := httptest.NewRequest(tkit.Request.Method, tkit.Request.Target, tkitReqBody)
			for hName, hValue := range tkit.Request.Headers {
				req.Header.Set(hName, hValue)
			}
			if tkit.Request.HeadersMakeFunc != nil {
				for hName, hValue := range tkit.Request.HeadersMakeFunc(t, &tkit) {
					req.Header.Set(hName, hValue)
				}
			}

			if tkit.BeforePerformRequestFunc != nil {
				tkit.BeforePerformRequestFunc(t, req, &tkit)
			}

			httpHandler.ServeHTTP(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			if tkit.AfterPerformRequestFunc != nil {
				tkit.AfterPerformRequestFunc(t, req, resp, &tkit)
			}

			assert.Exactly(t, tkit.Response.StatusCode, resp.StatusCode)
			for hName, hValue := range tkit.Response.Headers {
				assert.Exactly(t, hValue, resp.Header.Get(hName))
				req.Header.Set(hName, hValue)
			}

			respBodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			respBodyString := string(respBodyBytes)

			if tkit.Response.Body != "" {
				assert.Exactly(t, tkit.Response.Body, respBodyString)
			}

			if tkit.Response.BodyCheckFunc != nil {
				assert.Exactly(t, true, tkit.Response.BodyCheckFunc(t, respBodyString, &tkit))
			}

		})
	}
}

func HTTPTestJSONEncode(t *testing.T, v any) (ret string) {
	retBytes, err := json.Marshal(v)
	require.NoError(t, err)

	ret = string(retBytes)
	return
}
