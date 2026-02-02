package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

/* ============================================================
   Helpers
   ============================================================ */

func setupEcho() *echo.Echo {
	e := echo.New()
	api := e.Group("/api", AuthMiddleware)

	api.GET("/courses", GetCoursesHandler)
	api.GET("/courses/:courseId", GetCourseHandler)
	api.POST("/courses", CreateCourseHandler)
	api.PUT("/courses/:courseId", UpdateCourseHandler)

	return e
}

func authReq(method, path string, body []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid_token_123456")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func resetDB() {
	courseMu.Lock()
	defer courseMu.Unlock()

	courseDB = map[string]Course{
		"algorithms": {
			ID:           "algorithms",
			Name:         "Algorithms",
			Status:       CourseStatusCreated,
			StartDate:    "2024-01-01",
			EndDate:      "2024-02-01",
			RepoTemplate: "git@test/repo.git",
			Description:  "test",
			URL:          "/course/algorithms",
		},
		"hidden": {
			ID:           "hidden",
			Name:         "Hidden",
			Status:       CourseStatusHidden,
			StartDate:    "2024-01-01",
			EndDate:      "2024-02-01",
			RepoTemplate: "git@test/repo.git",
			Description:  "hidden",
			URL:          "/course/hidden",
		},
	}
}

/* ============================================================
   Validators (unit)
   ============================================================ */
func TestValidators(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() bool
		expected bool
	}{
		{"valid status", func() bool { return isValidCourseStatus("created") }, true},
		{"invalid status", func() bool { return isValidCourseStatus("broken") }, false},

		{"valid slug", func() bool { return isValidSlug("go-course") }, true},
		{"invalid slug uppercase", func() bool { return isValidSlug("Go-Course") }, false},
		{"invalid slug underscore", func() bool { return isValidSlug("go_course") }, false},
		{"invalid slug leading dash", func() bool { return isValidSlug("-go") }, false},
		{"invalid slug trailing dash", func() bool { return isValidSlug("go-") }, false},
		{"invalid slug double dash", func() bool { return isValidSlug("go--course") }, false},

		{"valid token", func() bool { return isValidToken("abcDEF123._-") }, true},
		{"invalid token space", func() bool { return isValidToken("bad token") }, false},

		{"valid date", func() bool { return isValidDate("2024-01-01") }, true},
		{"invalid date format", func() bool { return isValidDate("01-01-2024") }, false},

		{"valid date range", func() bool { return isValidDateRange("2024-01-01", "2024-01-02") }, true},
		{"invalid date range", func() bool { return isValidDateRange("2024-01-02", "2024-01-01") }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

/* ============================================================
   AuthMiddleware
   ============================================================ */

func TestAuthMiddleware_NoHeader(t *testing.T) {
	e := setupEcho()
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	e := setupEcho()
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	req.Header.Set("Authorization", "Bad token")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401")
	}
}

func TestAuthMiddleware_ShortToken(t *testing.T) {
	e := setupEcho()
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	req.Header.Set("Authorization", "Bearer short")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401")
	}
}

func TestAuthMiddleware_LongToken(t *testing.T) {
	e := setupEcho()
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	longToken := make([]byte, MaxTokenLength+1)
	for i := range longToken {
		longToken[i] = 'a'
	}
	req.Header.Set("Authorization", "Bearer "+string(longToken))
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidTokenChars(t *testing.T) {
	e := setupEcho()
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	req.Header.Set("Authorization", "Bearer токен123 space!")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_Success(t *testing.T) {
	e := echo.New()
	h := AuthMiddleware(func(c echo.Context) error {
		token := c.Get("token").(string)
		if token != "valid_token_123456" {
			t.Fatal("token not set correctly")
		}
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid_token_123456")
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	_ = h(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

/* ============================================================
   GET /courses
   ============================================================ */


func TestGetCourses_EmptyFilterResult(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodGet, "/api/courses?status=finished", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var courses []Course
	json.Unmarshal(rec.Body.Bytes(), &courses)

	if len(courses) != 0 {
		t.Fatalf("expected 0 courses, got %d", len(courses))
	}
}

func TestGetCourses(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodGet, "/api/courses", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
}

func TestGetCourses_Filter(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodGet, "/api/courses?status=hidden", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	var courses []Course
	_ = json.Unmarshal(rec.Body.Bytes(), &courses)

	if len(courses) != 1 {
		t.Fatalf("expected 1 filtered course")
	}
}

func TestGetCourses_NoFilterAllVisible(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodGet, "/api/courses", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	var courses []Course
	_ = json.Unmarshal(rec.Body.Bytes(), &courses)

	if len(courses) != 2 {
		t.Fatalf("expected 2 courses without filter")
	}
}

/* ============================================================
   GET /courses/:id
   ============================================================ */

func TestGetCourse_OK(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodGet, "/api/courses/algorithms", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
}

func TestGetCourse_NotFound(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodGet, "/api/courses/unknown", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404")
	}
}

/* ============================================================
   POST /courses
   ============================================================ */

func TestCreateCourse_EmptyRepoTemplate(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"name":"Test",
		"slug":"test",
		"status":"created",
		"startDate":"2025-01-01",
		"endDate":"2025-02-01",
		"description":"x"
	}`)

	req := authReq(http.MethodPost, "/api/courses", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateCourse_Success(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"name":"Go Course",
		"slug":"go-course",
		"status":"created",
		"startDate":"2024-03-01",
		"endDate":"2024-04-01",
		"repoTemplate":"git@test/go.git",
		"description":"Go basics"
	}`)

	req := authReq(http.MethodPost, "/api/courses", body)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestCreateCourse_ValidationError(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodPost, "/api/courses", []byte(`{"slug":"a"}`))
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
}

func TestCreateCourse_Conflict(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"name":"Algorithms",
		"slug":"algorithms",
		"status":"created",
		"startDate":"2024-01-01",
		"endDate":"2024-02-01",
		"repoTemplate":"git@test",
		"description":"dup"
	}`)

	req := authReq(http.MethodPost, "/api/courses", body)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409")
	}
}

func TestCreateCourse_InvalidDateRange(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"name":"Bad",
		"slug":"bad-course",
		"status":"created",
		"startDate":"2024-02-01",
		"endDate":"2024-01-01",
		"repoTemplate":"git@test",
		"description":"x"
	}`)

	req := authReq(http.MethodPost, "/api/courses", body)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
}

func TestCreateCourse_MissingRequiredFields(t *testing.T) {
	resetDB()
	e := setupEcho()

	cases := []struct {
		name         string
		body         string
		wantErrField string
	}{
		{"no name", `{"slug":"test","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@a","description":"x"}`, "name"},
		{"no slug", `{"name":"Test","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@a","description":"x"}`, "slug"},
		{"no status", `{"name":"Test","slug":"test","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@a","description":"x"}`, "status"},
		{"no repoTemplate", `{"name":"Test","slug":"test","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","description":"x"}`, "repoTemplate"},
		{"no description", `{"name":"Test","slug":"test","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@a"}`, "description"},
		{"no startDate", `{"name":"Test","slug":"test","status":"created","endDate":"2025-02-01","repoTemplate":"git@a","description":"x"}`, "startDate"},
		{"no endDate", `{"name":"Test","slug":"test","status":"created","startDate":"2025-01-01","repoTemplate":"git@a","description":"x"}`, "endDate"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := authReq(http.MethodPost, "/api/courses", []byte(tc.body))
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}

			var resp map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &resp)
			details, ok := resp["details"].([]interface{})
			if !ok {
				t.Fatal("expected details array")
			}
			found := false
			for _, d := range details {
				errMap := d.(map[string]interface{})
				if errMap["field"] == tc.wantErrField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected validation error for field %q", tc.wantErrField)
			}
		})
	}
}

func TestCreateCourse_InvalidSlugs(t *testing.T) {
	resetDB()
	e := setupEcho()

	badSlugs := []string{
		"Ab",           // uppercase + short
		"go_course",    // underscore
		"go course",    // space
		"го-курс",      // cyrillic
		"--a",          // leading dashes
		"a--",          // trailing dashes
		"a-b-c-d-e-f-g-h-i-j-k-l-m-n-o-p-q-r-s-t-u-v-w-x-y-z-0-1-2-3-4-5-6-7-8-9-0", // >50 chars
		"ab",           // too short
	}

	for _, slug := range badSlugs {
		t.Run(slug, func(t *testing.T) {
			body := fmt.Sprintf(`{"name":"Test","slug":"%s","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@a","description":"x"}`, slug)
			req := authReq(http.MethodPost, "/api/courses", []byte(body))
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestCreateCourse_InvalidDates(t *testing.T) {
	resetDB()
	e := setupEcho()

	badDates := []struct {
		name      string
		startDate string
		endDate   string
	}{
		{"invalid start format", "01-01-2025", "2025-02-01"},
		{"invalid end format", "2025-01-01", "01-02-2025"},
		{"end before start", "2025-02-01", "2025-01-01"},
	}

	for _, tc := range badDates {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"name":"Test","slug":"test","status":"created","startDate":"%s","endDate":"%s","repoTemplate":"git@a","description":"x"}`, tc.startDate, tc.endDate)
			req := authReq(http.MethodPost, "/api/courses", []byte(body))
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestCreateCourse_LongDescription(t *testing.T) {
	resetDB()
	e := setupEcho()

	longDesc := make([]byte, MaxDescLength+1)
	for i := range longDesc {
		longDesc[i] = 'a'
	}

	body := fmt.Sprintf(`{"name":"Test","slug":"test","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@a","description":"%s"}`, string(longDesc))
	req := authReq(http.MethodPost, "/api/courses", []byte(body))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateCourse_InvalidStatus(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{"name":"Test","slug":"test","status":"invalid","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@a","description":"x"}`)
	req := authReq(http.MethodPost, "/api/courses", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateCourse_InvalidJSON(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{ "name": "test"`) // malformed
	req := authReq(http.MethodPost, "/api/courses", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateCourse_ExtraFieldsIgnored(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"name":"Extra",
		"slug":"extra",
		"status":"created",
		"startDate":"2024-03-01",
		"endDate":"2024-04-01",
		"repoTemplate":"git@test",
		"description":"test",
		"id":"should-ignore",
		"url":"should-ignore"
	}`)

	req := authReq(http.MethodPost, "/api/courses", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var course Course
	json.Unmarshal(rec.Body.Bytes(), &course)
	if course.ID != "extra" {
		t.Error("ID should be set from slug")
	}
	if course.URL != "/course/extra" {
		t.Error("URL should be generated")
	}
}

/* ============================================================
   PUT /courses/:id
   ============================================================ */

func TestUpdateCourse_UpdateRepoTemplate(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{"repoTemplate":"git@updated"}`)
	req := authReq(http.MethodPut, "/api/courses/algorithms", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var updated Course
	json.Unmarshal(rec.Body.Bytes(), &updated)

	if updated.RepoTemplate != "git@updated" {
		t.Fatalf("repoTemplate not updated")
	}
}

func TestUpdateCourse_DateRangeValidAfterPartial(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{"endDate":"2024-03-01"}`)
	req := authReq(http.MethodPut, "/api/courses/algorithms", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUpdateCourse_AllFields(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"name":"Updated",
		"status":"finished",
		"startDate":"2024-01-10",
		"endDate":"2024-02-10",
		"repoTemplate":"git@new",
		"description":"updated"
	}`)

	req := authReq(http.MethodPut, "/api/courses/algorithms", body)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
}

func TestUpdateCourse_InvalidStatus(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodPut, "/api/courses/algorithms", []byte(`{"status":"bad"}`))
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400")
	}
}

func TestUpdateCourse_NotFound(t *testing.T) {
	resetDB()
	e := setupEcho()

	req := authReq(http.MethodPut, "/api/courses/unknown", []byte(`{"name":"x"}`))
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404")
	}
}

func TestUpdateCourse_PartialUpdate(t *testing.T) {
    resetDB()
    e := setupEcho()

    // Берём исходный статус до обновления
    courseMu.RLock()
    original := courseDB["algorithms"]
    courseMu.RUnlock()

    body := []byte(`{
        "name": "New Name Only",
        "description": "New desc only"
    }`)

    req := authReq(http.MethodPut, "/api/courses/algorithms", body)
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var updated Course
    json.Unmarshal(rec.Body.Bytes(), &updated)

    if updated.Name != "New Name Only" {
        t.Errorf("name not updated, got %q", updated.Name)
    }
    if updated.Description != "New desc only" {
        t.Errorf("description not updated, got %q", updated.Description)
    }
    if updated.Status != original.Status {
        t.Errorf("status changed unexpectedly: %q → %q", original.Status, updated.Status)
    }
    if updated.StartDate != original.StartDate {
        t.Error("startDate should not change")
    }
}

func TestUpdateCourse_EmptyFieldsIgnored(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"name":"",
		"status":"",
		"description":""
	}`)

	req := authReq(http.MethodPut, "/api/courses/algorithms", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var updated Course
	json.Unmarshal(rec.Body.Bytes(), &updated)

	if updated.Name == "" {
		t.Error("name should not be updated to empty")
	}
	if updated.Status == "" {
		t.Error("status should not be updated to empty")
	}
	if updated.Description == "" {
		t.Error("description should not be updated to empty")
	}
}

func TestUpdateCourse_InvalidDateRange(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{
		"startDate": "2025-03-01",
		"endDate":   "2025-02-01"
	}`)

	req := authReq(http.MethodPut, "/api/courses/algorithms", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateCourse_InvalidDateFormat(t *testing.T) {
	resetDB()
	e := setupEcho()

	cases := []string{
		`{"startDate": "01-03-2025"}`,
		`{"endDate": "01-04-2025"}`,
	}

	for _, bodyStr := range cases {
		req := authReq(http.MethodPut, "/api/courses/algorithms", []byte(bodyStr))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	}
}

func TestUpdateCourse_LongDescription(t *testing.T) {
	resetDB()
	e := setupEcho()

	longDesc := make([]byte, MaxDescLength+1)
	for i := range longDesc {
		longDesc[i] = 'a'
	}

	body := fmt.Sprintf(`{"description":"%s"}`, string(longDesc))
	req := authReq(http.MethodPut, "/api/courses/algorithms", []byte(body))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateCourse_InvalidNameLength(t *testing.T) {
	resetDB()
	e := setupEcho()

	cases := []string{
		`{"name": "ab"}`, // too short
	}

	longName := make([]byte, MaxNameLength+1)
	for i := range longName {
		longName[i] = 'a'
	}
	cases = append(cases, fmt.Sprintf(`{"name": "%s"}`, string(longName))) // too long

	for _, bodyStr := range cases {
		req := authReq(http.MethodPut, "/api/courses/algorithms", []byte(bodyStr))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	}
}

func TestUpdateCourse_IgnoreSlugChange(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{"slug": "new-slug"}`)

	req := authReq(http.MethodPut, "/api/courses/algorithms", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	courseMu.RLock()
	course := courseDB["algorithms"]
	courseMu.RUnlock()

	if course.ID != "algorithms" {
		t.Error("ID should not change")
	}
}

func TestUpdateCourse_InvalidJSON(t *testing.T) {
	resetDB()
	e := setupEcho()

	body := []byte(`{ "name": "test"`) // malformed
	req := authReq(http.MethodPut, "/api/courses/algorithms", body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateCourse_DoubleHyphenInSlug(t *testing.T) {
    resetDB()
    e := setupEcho()

    body := []byte(`{"name":"Test","slug":"go--course","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@test","description":"x"}`)
    req := authReq(http.MethodPost, "/api/courses", body)
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)

    if rec.Code != http.StatusBadRequest {
        t.Errorf("expected 400 for double hyphen, got %d", rec.Code)
    }
}

func TestCreateCourse_LeadingTrailingHyphen(t *testing.T) {
    resetDB()
    e := setupEcho()

    cases := []string{"-start", "end-", "-"}
    for _, slug := range cases {
        t.Run(slug, func(t *testing.T) {
            body := fmt.Sprintf(`{"name":"Test","slug":"%s","status":"created","startDate":"2025-01-01","endDate":"2025-02-01","repoTemplate":"git@test","description":"x"}`, slug)
            req := authReq(http.MethodPost, "/api/courses", []byte(body))
            rec := httptest.NewRecorder()
            e.ServeHTTP(rec, req)
            if rec.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for %q, got %d", slug, rec.Code)
            }
        })
    }
}

func TestPostCourseRequest_Validate_AllErrors(t *testing.T) {
	req := PostCourseRequest{
		Name:         "ab",              // too short
		Slug:         "BAD_slug",         // invalid
		Status:       "wrong",            // invalid
		StartDate:    "bad-date",
		EndDate:      "also-bad",
		RepoTemplate: "",
		Description:  string(make([]byte, MaxDescLength+1)),
	}

	errs := req.Validate()
	if len(errs) < 6 {
		t.Fatalf("expected many validation errors, got %d", len(errs))
	}
}

func TestIsValidDateRange_EqualDates(t *testing.T) {
	if isValidDateRange("2024-01-01", "2024-01-01") {
		t.Fatal("expected false when dates are equal")
	}
}
