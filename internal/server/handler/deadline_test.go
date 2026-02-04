package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func setupEchoForBoard() *echo.Echo {
	e := echo.New()
	e.GET("/api/courses/:courseId/board", GetCourseBoardHandler)
	return e
}

// ensureCourseInDB гарантирует, что курс существует в courseDB
func ensureCourseInDB(id, name string, status CourseStatus) {
	courseMu.Lock()
	defer courseMu.Unlock()
	if _, exists := courseDB[id]; !exists {
		courseDB[id] = Course{
			ID:           id,
			Name:         name,
			Status:       status,
			StartDate:    "2024-01-01",
			EndDate:      "2024-12-31",
			RepoTemplate: "git@test/repo.git",
			Description:  "test",
			URL:          "/course/" + id,
		}
	}
}

/* ============================================================
   Тесты для GetCourseBoardHandler
   ============================================================ */

func TestGetCourseBoardHandler_ValidCourse(t *testing.T) {
	// Убедимся, что оба курса существуют
	ensureCourseInDB("algorithms", "Algorithms 101", CourseStatusInProgress)
	ensureCourseInDB("mlops", "MLOps Studio", CourseStatusAllTasksIssued)

	boardData = map[string]TaskBoardSummary{
		"algorithms": {
			CourseName:    "Algorithms 101",
			CourseStatus:  "in_progress",
			SolvedScore:   126,
			MaxScore:      200,
			SolvedPercent: 63,
			Groups: []BoardGroup{
				{
					ID:        "week-1",
					Name:      "Week 1: Warmup",
					StartedAt: "2024-10-01T09:00:00Z",
					EndsAt:    "2024-10-14T18:00:00Z",
					Deadlines: []BoardDeadline{
						{ID: "d1", Label: "Checkpoint", Percent: 0.6, DueAt: "2024-09-20T18:00:00Z", Status: "expired"},
					},
					Tasks: []BoardTask{
						{ID: "t1", Name: "Arrays Sprint", Score: 20, ScoreEarned: 20, Stats: 0.82},
					},
				},
			},
		},
	}

	e := setupEchoForBoard()

	req := httptest.NewRequest(http.MethodGet, "/api/courses/algorithms/board", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp TaskBoardSummary
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Algorithms 101", resp.CourseName)
	assert.Equal(t, "in_progress", resp.CourseStatus)
	assert.Equal(t, 126, resp.SolvedScore)
	assert.Len(t, resp.Groups, 1)
	assert.Equal(t, "week-1", resp.Groups[0].ID)
	assert.Equal(t, "Checkpoint", resp.Groups[0].Deadlines[0].Label)
}

func TestGetCourseBoardHandler_CourseWithoutBoard(t *testing.T) {
	ensureCourseInDB("rust", "Rust Core", CourseStatusCreated)

	e := setupEchoForBoard()

	req := httptest.NewRequest(http.MethodGet, "/api/courses/rust/board", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp TaskBoardSummary
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Rust Core", resp.CourseName)
	assert.Equal(t, "created", resp.CourseStatus)
	assert.Empty(t, resp.Groups)
}

func TestGetCourseBoardHandler_CourseNotFound(t *testing.T) {
	e := setupEchoForBoard()

	req := httptest.NewRequest(http.MethodGet, "/api/courses/nonexistent/board", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "course not found", resp["error"])
}

func TestGetCourseBoardHandler_InvalidCourseID(t *testing.T) {
	ensureCourseInDB("algorithms", "Algorithms", CourseStatusInProgress)

	e := setupEchoForBoard()

	invalidIDs := []string{
		"",
		"AB",
		"go_course",
		"-start",
		"end-",
		"a",
		"toolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolong",
	}

	for _, id := range invalidIDs {
		t.Run("ID_"+id, func(t *testing.T) {
			url := "/api/courses/" + id + "/board"
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var resp map[string]string
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, "invalid course ID format", resp["error"])
		})
	}
}

func TestGetCourseBoardHandler_JSONStructure(t *testing.T) {
	ensureCourseInDB("mlops", "MLOps Studio", CourseStatusAllTasksIssued)

	boardData = map[string]TaskBoardSummary{
		"mlops": {
			CourseName:    "MLOps Studio",
			CourseStatus:  "all_tasks_issued",
			SolvedScore:   95,
			MaxScore:      150,
			SolvedPercent: 63,
			Groups: []BoardGroup{
				{
					ID:        "project-phase-1",
					Name:      "Project Phase 1",
					StartedAt: "2024-09-01T09:00:00Z",
					EndsAt:    "2024-10-15T18:00:00Z",
					Deadlines: []BoardDeadline{
						{ID: "mlops-d1", Label: "Proposal", Percent: 0.3, DueAt: "2024-09-15T18:00:00Z", Status: "expired"},
					},
					Tasks: []BoardTask{
						{ID: "mlops-t1", Name: "Data Pipeline", Score: 50, ScoreEarned: 45, Stats: 0.9},
					},
				},
			},
		},
	}

	e := setupEchoForBoard()

	req := httptest.NewRequest(http.MethodGet, "/api/courses/mlops/board", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)

	requiredFields := []string{"courseName", "courseStatus", "solvedScore", "maxScore", "solvedPercent", "groups"}
	for _, field := range requiredFields {
		assert.Contains(t, resp, field, "missing field: %s", field)
	}

	groups := resp["groups"].([]interface{})
	assert.GreaterOrEqual(t, len(groups), 1)

	group := groups[0].(map[string]interface{})
	assert.Contains(t, group, "id")
	assert.Contains(t, group, "name")
	assert.Contains(t, group, "deadlines")
	assert.Contains(t, group, "tasks")

	deadlines := group["deadlines"].([]interface{})
	deadline := deadlines[0].(map[string]interface{})
	assert.Contains(t, deadline, "id")
	assert.Contains(t, deadline, "label")
	assert.Contains(t, deadline, "percent")
	assert.Contains(t, deadline, "dueAt")
	assert.Contains(t, deadline, "status")
}