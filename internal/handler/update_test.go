package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricUpdate(t *testing.T) {

	type want struct {
		StatusCode int
	}
	tests := []struct {
		name    string
		request string
		want
	}{
		{
			name:    "successfull gauge metric update #1",
			request: "/update/gauge/HeapObjects/3522.0",
			want:    want{StatusCode: 200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(Update)
			h(w, request)

			result := w.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.want.StatusCode, result.StatusCode)
		})
	}

}
