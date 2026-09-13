package study

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

const adminPassword = "correct horse battery staple"

func adminTestServer(t *testing.T) (*Server, *Store) {
	t.Helper()
	store := openTest(t, 100)
	return NewServer(store, fstest.MapFS{}, WithAdmin(&AdminConfig{Username: "root", Password: adminPassword})), store
}

func adminCall(s *Server, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	s.ServeHTTP(response, request)
	return response
}

func adminLogin(t *testing.T, s *Server, password string) (*http.Cookie, *httptest.ResponseRecorder) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"username": "root", "password": password})
	if err != nil {
		t.Fatal(err)
	}
	response := adminCall(s, http.MethodPost, "/api/admin/session", string(body), nil)
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == adminCookieName {
			return cookie, response
		}
	}
	return nil, response
}

func TestLoadAdminConfig(t *testing.T) {
	directory := t.TempDir()
	missing := filepath.Join(directory, "absent.json")
	if config, err := LoadAdminConfig(missing); err != nil || config != nil {
		t.Fatalf("missing file should disable the panel: %v %v", config, err)
	}
	write := func(name, content string) string {
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for name, content := range map[string]string{
		"broken.json":    `{"username":`,
		"empty.json":     `{"username":"","password":""}`,
		"truncated.json": `{"username":"root","password":"secret"} extra`,
	} {
		if _, err := LoadAdminConfig(write(name, content)); err == nil {
			t.Fatalf("accepted invalid config %s", name)
		}
	}
	path := write("admin.json", `{"username":" root ","password":"secret"}`)
	config, err := LoadAdminConfig(path)
	if err != nil || config == nil || config.Username != "root" || config.Password != "secret" {
		t.Fatalf("unexpected config: %+v %v", config, err)
	}
}

func TestAdminPanelHiddenWithoutCredentials(t *testing.T) {
	server := NewServer(openTest(t, 10), fstest.MapFS{})
	for _, path := range []string{"/api/admin/session", "/api/admin/catalog", "/api/admin/submissions"} {
		response := adminCall(server, http.MethodGet, path, "", nil)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s exposed without credentials: %d", path, response.Code)
		}
	}
}

func TestAdminLoginRequiresTheConfiguredSecret(t *testing.T) {
	server, _ := adminTestServer(t)
	for _, password := range []string{"", "wrong", adminPassword + " "} {
		cookie, response := adminLogin(t, server, password)
		if response.Code != http.StatusUnauthorized || cookie != nil {
			t.Fatalf("accepted password %q: %d", password, response.Code)
		}
	}
	anonymous := adminCall(server, http.MethodGet, "/api/admin/submissions", "", nil)
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated read allowed: %d", anonymous.Code)
	}
	cookie, response := adminLogin(t, server, adminPassword)
	if response.Code != http.StatusOK || cookie == nil {
		t.Fatalf("login failed: %d", response.Code)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != adminCookiePath {
		t.Fatalf("weak session cookie: %+v", cookie)
	}
	authorized := adminCall(server, http.MethodGet, "/api/admin/submissions", "", cookie)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized read rejected: %d", authorized.Code)
	}
	if header := authorized.Header().Get("X-Robots-Tag"); header != "noindex, nofollow" {
		t.Fatalf("admin response is indexable: %q", header)
	}
}

func TestAdminLogoutRevokesTheSession(t *testing.T) {
	server, _ := adminTestServer(t)
	cookie, _ := adminLogin(t, server, adminPassword)
	if response := adminCall(server, http.MethodDelete, "/api/admin/session", "", cookie); response.Code != http.StatusOK {
		t.Fatalf("logout failed: %d", response.Code)
	}
	if response := adminCall(server, http.MethodGet, "/api/admin/catalog", "", cookie); response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked cookie still accepted: %d", response.Code)
	}
}

func TestAdminLoginThrottlesGuessing(t *testing.T) {
	server, _ := adminTestServer(t)
	for range adminLoginLimit {
		if _, response := adminLogin(t, server, "wrong"); response.Code != http.StatusUnauthorized {
			t.Fatalf("unexpected failure mode: %d", response.Code)
		}
	}
	cookie, response := adminLogin(t, server, adminPassword)
	if response.Code != http.StatusTooManyRequests || cookie != nil {
		t.Fatalf("guessing not throttled: %d", response.Code)
	}
}

func TestAdminSubmissionsExposeFullNormalizedRecords(t *testing.T) {
	server, store := adminTestServer(t)
	risk := 20
	records := []SurveySubmission{
		{Status: []string{"降智"}, Answers: map[string][]string{"plans": {"Plus"}, "usage": {"直登"}}, Details: map[string]string{"country": "PH", "concurrency": "8"}},
		{Status: []string{"风控（限流）", "封号"}, Answers: map[string][]string{"plans": {"Free 免费"}, "usage": {"反代"}}, Details: map[string]string{"country": "SG", "concurrency": "15", "duration": "45", "durationUnit": "天"}, IPRisk: &risk},
	}
	for _, record := range records {
		if err := store.putSurvey(record); err != nil {
			t.Fatal(err)
		}
	}
	cookie, _ := adminLogin(t, server, adminPassword)
	response := adminCall(server, http.MethodGet, "/api/admin/submissions", "", cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("read failed: %d", response.Code)
	}
	var payload adminSubmissions
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 2 || payload.Returned != 2 || payload.Truncated {
		t.Fatalf("unexpected envelope: %+v", payload)
	}
	newest, oldest := payload.Records[0], payload.Records[1]
	if newest.ID <= oldest.ID {
		t.Fatalf("records not newest-first: %d then %d", newest.ID, oldest.ID)
	}
	// The panel must show what was submitted. "SG" is a real answer, and
	// collapsing it into "其他国家或地区" alone would hide it from the operator.
	if got := strings.Join(newest.Raw["country"], ","); got != "SG" {
		t.Fatalf("submitted country code lost: %q", got)
	}
	if got := strings.Join(newest.Normalized["country"], ","); got != "其他国家或地区" {
		t.Fatalf("statistics bucket missing: %q", got)
	}
	if got := strings.Join(newest.Raw["concurrency"], ","); got != "15" {
		t.Fatalf("submitted concurrency lost: %q", got)
	}
	if got := strings.Join(newest.Normalized["concurrency"], ","); got != "11 及以上" {
		t.Fatalf("concurrency not bucketed: %q", got)
	}
	if got := strings.Join(newest.Raw["duration"], ","); got != "45 天" {
		t.Fatalf("submitted duration lost: %q", got)
	}
	if got := strings.Join(newest.Normalized["duration"], ","); got != "30–89 天" {
		t.Fatalf("duration not bucketed: %q", got)
	}
	if got := strings.Join(newest.Raw["ipRisk"], ","); got != "20" {
		t.Fatalf("submitted risk score lost: %q", got)
	}
	if got := strings.Join(newest.Normalized["ipRisk"], ","); got != "15–24" {
		t.Fatalf("numeric risk not bucketed: %q", got)
	}
	if !strings.Contains(strings.Join(newest.Normalized["plans"], ","), "Free 免费") {
		t.Fatal("raw plan answer missing from normalized view")
	}
	if got := strings.Join(oldest.Normalized["country"], ","); got != "菲律宾" {
		t.Fatalf("country code not normalized: %q", got)
	}
	if got := strings.Join(oldest.Raw["country"], ","); got != "PH" {
		t.Fatalf("submitted country code lost: %q", got)
	}
	if got := strings.Join(oldest.Normalized["concurrency"], ","); got != "6–10" {
		t.Fatalf("concurrency not bucketed: %q", got)
	}
	if got := strings.Join(oldest.Raw["concurrency"], ","); got != "8" {
		t.Fatalf("submitted concurrency lost: %q", got)
	}
	if newest.SubmittedAt == nil || oldest.SubmittedAt == nil {
		t.Fatal("server receive time missing")
	}
	if len(newest.Status) != 2 || oldest.Details["concurrency"] != "8" {
		t.Fatalf("raw detail lost: %+v %+v", newest.Status, oldest.Details)
	}
}

func TestAdminCatalogMatchesTheQuestionnaire(t *testing.T) {
	server, _ := adminTestServer(t)
	cookie, _ := adminLogin(t, server, adminPassword)
	response := adminCall(server, http.MethodGet, "/api/admin/catalog", "", cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("catalog read failed: %d", response.Code)
	}
	var catalog surveyCatalogData
	if err := json.Unmarshal(response.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Definitions) != len(surveyCatalog.Definitions) {
		t.Fatalf("question count changed: %d", len(catalog.Definitions))
	}
	for _, definition := range catalog.Definitions {
		if definition.Key == "country" {
			if !strings.Contains(strings.Join(catalog.Choices["country"], ","), "菲律宾") {
				t.Fatal("country choices missing")
			}
			return
		}
	}
	t.Fatal("country question missing from catalog")
}
