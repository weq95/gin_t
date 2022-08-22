package test

import (
	"gin-core/core"
	"net/http"
	"net/http/httptest"
	"testing"
)

func runRequest(B *testing.B, r *core.Engine, method, path string) {
	// create fake request
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	w := httptest.NewRecorder()
	B.ReportAllocs()
	B.ResetTimer()
	for i := 0; i < B.N; i++ {
		r.ServeHTTP(w, req)
	}

}

func BenchmarkJson(b *testing.B) {
	for i := 0; i < b.N; i++ {
		router := core.New()
		router.GET("/json", func(c *core.Context) { c.JSON(map[string]any{"hello": "world"}) })
		router.TestInit()
		runRequest(b, router, "GET", "/json")
	}
}
