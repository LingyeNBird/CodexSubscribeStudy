package study

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func annotationCall(t *testing.T, s *Server, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	return adminCall(s, http.MethodPut, "/api/admin/annotations", body, cookie)
}

func countAnnotations(t *testing.T, store *Store) int {
	t.Helper()
	var total int
	if err := store.db.View(func(tx *sql.Tx) error {
		return tx.QueryRow("SELECT COUNT(*) FROM survey_annotations").Scan(&total)
	}); err != nil {
		t.Fatal(err)
	}
	return total
}

func readAdminRecord(t *testing.T, s *Server, cookie *http.Cookie, index int) adminSubmission {
	t.Helper()
	response := adminCall(s, http.MethodGet, "/api/admin/submissions", "", cookie)
	var payload adminSubmissions
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if index >= len(payload.Records) {
		t.Fatalf("missing record %d", index)
	}
	return payload.Records[index]
}

func TestAnnotationsAreStoredWithoutTouchingTheSubmission(t *testing.T) {
	server, store := adminTestServer(t)
	if err := store.putSurvey(SurveySubmission{Status: []string{"降智"}, Answers: map[string][]string{"plans": {"Plus"}}}); err != nil {
		t.Fatal(err)
	}
	if err := store.putSurvey(SurveySubmission{Status: []string{"正常"}, Answers: map[string][]string{"plans": {"Go"}}}); err != nil {
		t.Fatal(err)
	}
	cookie, _ := adminLogin(t, server, adminPassword)
	if response := annotationCall(t, server, cookie, `{"ids":[1],"addTags":["可疑"],"note":"地区与时长对不上"}`); response.Code != http.StatusOK {
		t.Fatalf("annotation rejected: %d %s", response.Code, response.Body)
	}
	if response := annotationCall(t, server, cookie, `{"ids":[1,2],"addTags":["待跟进"]}`); response.Code != http.StatusOK {
		t.Fatalf("batch annotation rejected: %d %s", response.Code, response.Body)
	}
	first := readAdminRecord(t, server, cookie, 1)
	second := readAdminRecord(t, server, cookie, 0)
	if strings.Join(first.Tags, ",") != "可疑,待跟进" || first.Note != "地区与时长对不上" {
		t.Fatalf("annotation lost or reordered: %+v %q", first.Tags, first.Note)
	}
	if strings.Join(second.Tags, ",") != "待跟进" || second.Note != "" {
		t.Fatalf("batch annotation leaked: %+v", second)
	}
	// The submitted answer itself must stay exactly as received.
	if len(first.Answers["plans"]) != 1 || first.Answers["plans"][0] != "Plus" {
		t.Fatalf("submission mutated: %+v", first.Answers)
	}
}

func TestAnnotationRemovalAndClearing(t *testing.T) {
	server, store := adminTestServer(t)
	if err := store.putSurvey(SurveySubmission{Status: []string{"正常"}, Answers: map[string][]string{"plans": {"Plus"}}}); err != nil {
		t.Fatal(err)
	}
	cookie, _ := adminLogin(t, server, adminPassword)
	if response := annotationCall(t, server, cookie, `{"ids":[1],"addTags":["可疑","待跟进"],"note":"x"}`); response.Code != http.StatusOK {
		t.Fatalf("annotation rejected: %d", response.Code)
	}
	if response := annotationCall(t, server, cookie, `{"ids":[1],"removeTags":["待跟进"]}`); response.Code != http.StatusOK {
		t.Fatalf("removal rejected: %d", response.Code)
	}
	record := readAdminRecord(t, server, cookie, 0)
	if strings.Join(record.Tags, ",") != "可疑" || record.Note != "x" {
		t.Fatalf("removal damaged other fields: %+v", record)
	}
	// Clearing every field drops the row instead of leaving a blank annotation.
	if response := annotationCall(t, server, cookie, `{"ids":[1],"removeTags":["可疑"],"note":""}`); response.Code != http.StatusOK {
		t.Fatalf("clear rejected: %d", response.Code)
	}
	record = readAdminRecord(t, server, cookie, 0)
	if len(record.Tags) != 0 || record.Note != "" {
		t.Fatalf("annotation not cleared: %+v", record)
	}
	if total := countAnnotations(t, store); total != 0 {
		t.Fatalf("blank annotation kept: %d", total)
	}
}

func TestAnnotationRequestsAreValidated(t *testing.T) {
	server, store := adminTestServer(t)
	if err := store.putSurvey(SurveySubmission{Status: []string{"正常"}, Answers: map[string][]string{"plans": {"Plus"}}}); err != nil {
		t.Fatal(err)
	}
	cookie, _ := adminLogin(t, server, adminPassword)
	long := strings.Repeat("长", annotationTagLength+1)
	for _, body := range []string{
		`{}`,
		`{"ids":[]}`,
		`{"ids":[0],"addTags":["x"]}`,
		`{"ids":[1]}`,
		`{"ids":[1],"addTags":["` + long + `"]}`,
		`{"ids":[1],"note":"` + strings.Repeat("n", annotationNoteLength+1) + `"}`,
		`{"ids":[1],"addTags":["x"],"unknown":1}`,
		`{"ids":[1],"addTags":["x"]} trailing`,
	} {
		if response := annotationCall(t, server, cookie, body); response.Code != http.StatusBadRequest {
			t.Fatalf("accepted %s: %d", body, response.Code)
		}
	}
	if response := annotationCall(t, server, cookie, `{"ids":[1],"addTags":["x"]}`); response.Code != http.StatusOK {
		t.Fatalf("valid request rejected: %d", response.Code)
	}
	// Unknown submission ids are ignored rather than creating orphan rows.
	if response := annotationCall(t, server, cookie, `{"ids":[999],"addTags":["x"]}`); response.Code != http.StatusOK {
		t.Fatalf("unknown id rejected: %d", response.Code)
	}
	if total := countAnnotations(t, store); total != 1 {
		t.Fatalf("orphan annotation written: %d", total)
	}
}

func TestAnnotationRequiresTheAdminSession(t *testing.T) {
	server, store := adminTestServer(t)
	if err := store.putSurvey(SurveySubmission{Status: []string{"正常"}, Answers: map[string][]string{"plans": {"Plus"}}}); err != nil {
		t.Fatal(err)
	}
	if response := annotationCall(t, server, nil, `{"ids":[1],"addTags":["x"]}`); response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous write allowed: %d", response.Code)
	}
	if total := countAnnotations(t, store); total != 0 {
		t.Fatal("anonymous request wrote an annotation")
	}
}
