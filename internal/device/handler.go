package device

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServiceInterface interface {
	Create(ctx context.Context, in CreateInput) (*Device, error)
	Get(ctx context.Context, id string) (*Device, error)
	List(ctx context.Context, brand string, state State) ([]Device, error)
	Patch(ctx context.Context, id string, in PatchInput) (*Device, error)
	Delete(ctx context.Context, id string) error
}

type Handler struct {
	svc ServiceInterface
}

func NewHandler(svc ServiceInterface) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	r.POST("/devices", h.create)
	r.PATCH("/devices/:id", h.patch)
	r.GET("/devices/:id", h.get)
	r.GET("/devices", h.list)
	r.DELETE("/devices/:id", h.delete)
}

type createRequest struct {
	Name  string `json:"name" binding:"required" example:"iPhone 15 Pro"`
	Brand string `json:"brand" binding:"required" example:"Apple"`
	State State  `json:"state" enums:"available,in-use,inactive" example:"available"`
}

type patchRequest struct {
	Name  *string `json:"name" example:"iPhone 15 Pro Max"`
	Brand *string `json:"brand" example:"Apple Inc."`
	State *State  `json:"state" enums:"available,in-use,inactive" example:"in-use"`
}

// create godoc
// @Summary      Create a device
// @Description  Creates a new device with the given properties
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        device  body      createRequest  true  "Device creation data"
// @Success      201     {object}  Device
// @Failure      400     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /api/v1/devices [post]
func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	d, err := h.svc.Create(c.Request.Context(), CreateInput{
		Name:  req.Name,
		Brand: req.Brand,
		State: req.State,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Location", "/api/v1/devices/"+d.ID)
	c.JSON(http.StatusCreated, d)
}

// list godoc
// @Summary      List devices
// @Description  Retrieves a list of devices, optionally filtered by brand and/or state
// @Tags         devices
// @Produce      json
// @Param        brand  query     string  false  "Filter by exact brand name"
// @Param        state  query     string  false  "Filter by state (available, in-use, inactive)"
// @Success      200    {array}   Device
// @Failure      400    {object}  map[string]string
// @Failure      500    {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /api/v1/devices [get]
func (h *Handler) list(c *gin.Context) {
	devices, err := h.svc.List(c.Request.Context(), c.Query("brand"), State(c.Query("state")))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, devices)
}

// get godoc
// @Summary      Get a device
// @Description  Retrieves a single device by its ID
// @Tags         devices
// @Produce      json
// @Param        id   path      string  true  "Device ID (UUID)"
// @Success      200  {object}  Device
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /api/v1/devices/{id} [get]
func (h *Handler) get(c *gin.Context) {
	d, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

// patch godoc
// @Summary      Partially update a device
// @Description  Updates only the provided fields of a device
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        id      path      string        true  "Device ID (UUID)"
// @Param        device  body      patchRequest  true  "Device partial data"
// @Success      200     {object}  Device
// @Failure      400     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      409     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /api/v1/devices/{id} [patch]
func (h *Handler) patch(c *gin.Context) {
	var req patchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}
	d, err := h.svc.Patch(c.Request.Context(), c.Param("id"), PatchInput{
		Name:  req.Name,
		Brand: req.Brand,
		State: req.State,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

// delete godoc
// @Summary      Delete a device
// @Description  Deletes a device by its ID (only if not in-use)
// @Tags         devices
// @Param        id   path      string  true  "Device ID (UUID)"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /api/v1/devices/{id} [delete]
func (h *Handler) delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInUse):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidState):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
