package study

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestShareImagesAllowedWithoutBlobScripts(t *testing.T) {
	server := NewServer(openTest(t, 10), fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>Survey</title>")},
	})
	response := call(server, "/", http.MethodGet, nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("page returned %d", response.Code)
	}
	directives := map[string][]string{}
	for _, directive := range strings.Split(response.Header().Get("Content-Security-Policy"), ";") {
		fields := strings.Fields(directive)
		if len(fields) > 0 {
			directives[fields[0]] = fields[1:]
		}
	}
	if !slices.Contains(directives["img-src"], "blob:") {
		t.Fatal("share preview and PNG export cannot load their local blob images")
	}
	for _, directive := range []string{"default-src", "script-src", "connect-src"} {
		sources, ok := directives[directive]
		if !ok {
			t.Fatalf("missing %s restriction", directive)
		}
		for _, unsafe := range []string{"blob:", "data:", "*", "'unsafe-inline'", "'unsafe-eval'"} {
			if slices.Contains(sources, unsafe) {
				t.Errorf("image support unexpectedly permits %s in %s", unsafe, directive)
			}
		}
	}
}
