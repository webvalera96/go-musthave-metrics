package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func AddChiURLParams(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.NewRouteContext()
	for k, v := range params {
		ctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}

func TestMetricUpdate(t *testing.T) {

	type want struct {
		StatusCode int
	}
	tests := []struct {
		name      string
		request   string
		chiParams map[string]string
		want
	}{
		{
			name:    "successfull gauge metric update #1",
			request: "/update/gauge/HeapObjects/3522.0",
			chiParams: map[string]string{
				"metricType":  "gauge",
				"metricName":  "HeapObjects",
				"metricValue": "3522.0",
			},
			want: want{StatusCode: 200},
		},
		{
			name:    "successfull counter metric update #2",
			request: "/update/counter/PollCount/2",
			chiParams: map[string]string{
				"metricType":  "counter",
				"metricName":  "PollCount",
				"metricValue": "2",
			},
			want: want{StatusCode: 200},
		},
		{
			name:    "unsuccessfull no metric in url #3",
			request: "/update/gauge/3522.0",
			chiParams: map[string]string{
				"metricType":  "gauge",
				"metricName":  "",
				"metricValue": "3522.0",
			},
			want: want{StatusCode: 404},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
			request = AddChiURLParams(request, tt.chiParams)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(NewUpdateHandler().ServeHTTP)
			h(w, request)

			result := w.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.want.StatusCode, result.StatusCode)
		})
	}

}
