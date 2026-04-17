package device

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock service
// ---------------------------------------------------------------------------

type mockService struct {
	createFn func(ctx context.Context, in CreateInput) (*Device, error)
	getFn    func(ctx context.Context, id string) (*Device, error)
	listFn   func(ctx context.Context, brand string, state State) ([]Device, error)
	patchFn  func(ctx context.Context, id string, in PatchInput) (*Device, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockService) Create(ctx context.Context, in CreateInput) (*Device, error) {
	return m.createFn(ctx, in)
}

func (m *mockService) Get(ctx context.Context, id string) (*Device, error) {
	return m.getFn(ctx, id)
}

func (m *mockService) List(ctx context.Context, brand string, state State) ([]Device, error) {
	return m.listFn(ctx, brand, state)
}

func (m *mockService) Patch(ctx context.Context, id string, in PatchInput) (*Device, error) {
	return m.patchFn(ctx, id, in)
}

func (m *mockService) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var fixedTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func sampleDevice() *Device {
	return &Device{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Name:         "iPhone 15",
		Brand:        "Apple",
		State:        StateAvailable,
		CreationTime: fixedTime,
	}
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	h.Register(g)
	return r
}

func doRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	return m
}

// ===========================================================================
// POST /api/v1/devices — create
// ===========================================================================

func TestCreate_Success(t *testing.T) {
	d := sampleDevice()
	mock := &mockService{
		createFn: func(_ context.Context, in CreateInput) (*Device, error) {
			assert.Equal(t, "iPhone 15", in.Name)
			assert.Equal(t, "Apple", in.Brand)
			return d, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPost, "/api/v1/devices", map[string]string{
		"name":  "iPhone 15",
		"brand": "Apple",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Header().Get("Location"), d.ID)

	body := parseBody(t, w)
	assert.Equal(t, d.ID, body["id"])
	assert.Equal(t, "iPhone 15", body["name"])
	assert.Equal(t, "Apple", body["brand"])
}

func TestCreate_MissingName(t *testing.T) {
	mock := &mockService{}
	h := NewHandler(mock)
	r := setupRouter(h)

	w := doRequest(r, http.MethodPost, "/api/v1/devices", map[string]string{
		"brand": "Apple",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := parseBody(t, w)
	assert.Contains(t, body, "error")
}

func TestCreate_MissingBrand(t *testing.T) {
	mock := &mockService{}
	h := NewHandler(mock)
	r := setupRouter(h)

	w := doRequest(r, http.MethodPost, "/api/v1/devices", map[string]string{
		"name": "iPhone 15",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_EmptyBody(t *testing.T) {
	mock := &mockService{}
	h := NewHandler(mock)
	r := setupRouter(h)

	w := doRequest(r, http.MethodPost, "/api/v1/devices", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_InvalidState(t *testing.T) {
	mock := &mockService{
		createFn: func(_ context.Context, _ CreateInput) (*Device, error) {
			return nil, ErrInvalidState
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPost, "/api/v1/devices", map[string]string{
		"name":  "iPhone 15",
		"brand": "Apple",
		"state": "broken",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_InternalError(t *testing.T) {
	mock := &mockService{
		createFn: func(_ context.Context, _ CreateInput) (*Device, error) {
			return nil, errors.New("db connection lost")
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPost, "/api/v1/devices", map[string]string{
		"name":  "iPhone 15",
		"brand": "Apple",
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := parseBody(t, w)
	assert.Equal(t, "internal error", body["error"])
}

// ===========================================================================
// GET /api/v1/devices/:id — get
// ===========================================================================

func TestGet_Success(t *testing.T) {
	d := sampleDevice()
	mock := &mockService{
		getFn: func(_ context.Context, id string) (*Device, error) {
			assert.Equal(t, d.ID, id)
			return d, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices/"+d.ID, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseBody(t, w)
	assert.Equal(t, d.ID, body["id"])
}

func TestGet_NotFound(t *testing.T) {
	mock := &mockService{
		getFn: func(_ context.Context, _ string) (*Device, error) {
			return nil, ErrNotFound
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGet_InvalidInput(t *testing.T) {
	mock := &mockService{
		getFn: func(_ context.Context, _ string) (*Device, error) {
			return nil, ErrInvalidInput
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices/not-a-uuid", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGet_InternalError(t *testing.T) {
	mock := &mockService{
		getFn: func(_ context.Context, _ string) (*Device, error) {
			return nil, errors.New("unexpected")
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := parseBody(t, w)
	assert.Equal(t, "internal error", body["error"])
}

// ===========================================================================
// GET /api/v1/devices — list
// ===========================================================================

func TestList_Success(t *testing.T) {
	devices := []Device{*sampleDevice()}
	mock := &mockService{
		listFn: func(_ context.Context, brand string, state State) ([]Device, error) {
			assert.Empty(t, brand)
			assert.Empty(t, string(state))
			return devices, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	assert.Equal(t, "iPhone 15", result[0]["name"])
}

func TestList_WithBrandFilter(t *testing.T) {
	mock := &mockService{
		listFn: func(_ context.Context, brand string, state State) ([]Device, error) {
			assert.Equal(t, "Apple", brand)
			return []Device{}, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices?brand=Apple", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_WithStateFilter(t *testing.T) {
	mock := &mockService{
		listFn: func(_ context.Context, brand string, state State) ([]Device, error) {
			assert.Equal(t, StateAvailable, state)
			return []Device{}, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices?state=available", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_WithBothFilters(t *testing.T) {
	mock := &mockService{
		listFn: func(_ context.Context, brand string, state State) ([]Device, error) {
			assert.Equal(t, "Samsung", brand)
			assert.Equal(t, StateInUse, state)
			return []Device{}, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices?brand=Samsung&state=in-use", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_EmptyResult(t *testing.T) {
	mock := &mockService{
		listFn: func(_ context.Context, _ string, _ State) ([]Device, error) {
			return []Device{}, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Empty(t, result)
}

func TestList_InvalidState(t *testing.T) {
	mock := &mockService{
		listFn: func(_ context.Context, _ string, _ State) ([]Device, error) {
			return nil, ErrInvalidState
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices?state=broken", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_InternalError(t *testing.T) {
	mock := &mockService{
		listFn: func(_ context.Context, _ string, _ State) ([]Device, error) {
			return nil, errors.New("db error")
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodGet, "/api/v1/devices", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ===========================================================================
// PATCH /api/v1/devices/:id — patch
// ===========================================================================

func TestPatch_Success(t *testing.T) {
	d := sampleDevice()
	d.Name = "iPhone 16"
	mock := &mockService{
		patchFn: func(_ context.Context, id string, in PatchInput) (*Device, error) {
			assert.Equal(t, d.ID, id)
			require.NotNil(t, in.Name)
			assert.Equal(t, "iPhone 16", *in.Name)
			return d, nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPatch, "/api/v1/devices/"+d.ID, map[string]string{
		"name": "iPhone 16",
	})

	assert.Equal(t, http.StatusOK, w.Code)
	body := parseBody(t, w)
	assert.Equal(t, "iPhone 16", body["name"])
}

func TestPatch_EmptyBody(t *testing.T) {
	mock := &mockService{}
	h := NewHandler(mock)
	r := setupRouter(h)

	w := doRequest(r, http.MethodPatch, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatch_NotFound(t *testing.T) {
	mock := &mockService{
		patchFn: func(_ context.Context, _ string, _ PatchInput) (*Device, error) {
			return nil, ErrNotFound
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPatch, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", map[string]string{
		"name": "X",
	})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPatch_InUse(t *testing.T) {
	mock := &mockService{
		patchFn: func(_ context.Context, _ string, _ PatchInput) (*Device, error) {
			return nil, ErrInUse
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPatch, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", map[string]string{
		"name": "X",
	})

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPatch_InvalidInput(t *testing.T) {
	mock := &mockService{
		patchFn: func(_ context.Context, _ string, _ PatchInput) (*Device, error) {
			return nil, ErrInvalidInput
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPatch, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", map[string]string{
		"name": "",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatch_InvalidState(t *testing.T) {
	mock := &mockService{
		patchFn: func(_ context.Context, _ string, _ PatchInput) (*Device, error) {
			return nil, ErrInvalidState
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPatch, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", map[string]string{
		"state": "broken",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPatch_InternalError(t *testing.T) {
	mock := &mockService{
		patchFn: func(_ context.Context, _ string, _ PatchInput) (*Device, error) {
			return nil, errors.New("unexpected")
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodPatch, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", map[string]string{
		"name": "X",
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := parseBody(t, w)
	assert.Equal(t, "internal error", body["error"])
}

// ===========================================================================
// DELETE /api/v1/devices/:id — delete
// ===========================================================================

func TestDelete_Success(t *testing.T) {
	mock := &mockService{
		deleteFn: func(_ context.Context, id string) error {
			assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id)
			return nil
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodDelete, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestDelete_NotFound(t *testing.T) {
	mock := &mockService{
		deleteFn: func(_ context.Context, _ string) error {
			return ErrNotFound
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodDelete, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_InUse(t *testing.T) {
	mock := &mockService{
		deleteFn: func(_ context.Context, _ string) error {
			return ErrInUse
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodDelete, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDelete_InternalError(t *testing.T) {
	mock := &mockService{
		deleteFn: func(_ context.Context, _ string) error {
			return errors.New("disk full")
		},
	}

	h := NewHandler(mock)
	r := setupRouter(h)
	w := doRequest(r, http.MethodDelete, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := parseBody(t, w)
	assert.Equal(t, "internal error", body["error"])
}

// ===========================================================================
// writeError mapping — exhaustive coverage
// ===========================================================================

func TestWriteError_MapsAllSentinelErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"ErrNotFound", ErrNotFound, http.StatusNotFound, "device not found"},
		{"ErrInUse", ErrInUse, http.StatusConflict, "device is in use"},
		{"ErrInvalidInput", ErrInvalidInput, http.StatusBadRequest, "invalid input"},
		{"ErrInvalidState", ErrInvalidState, http.StatusBadRequest, "invalid device state"},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError, "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{
				getFn: func(_ context.Context, _ string) (*Device, error) {
					return nil, tt.err
				},
			}

			h := NewHandler(mock)
			r := setupRouter(h)
			w := doRequest(r, http.MethodGet, "/api/v1/devices/550e8400-e29b-41d4-a716-446655440000", nil)

			assert.Equal(t, tt.wantStatus, w.Code)
			body := parseBody(t, w)
			assert.Equal(t, tt.wantBody, body["error"])
		})
	}
}
