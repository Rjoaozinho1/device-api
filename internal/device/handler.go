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
	Name  string `json:"name" binding:"required"`
	Brand string `json:"brand" binding:"required"`
	State State  `json:"state"`
}

type patchRequest struct {
	Name  *string `json:"name"`
	Brand *string `json:"brand"`
	State *State  `json:"state"`
}

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

func (h *Handler) list(c *gin.Context) {
	devices, err := h.svc.List(c.Request.Context(), c.Query("brand"), State(c.Query("state")))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, devices)
}

func (h *Handler) get(c *gin.Context) {
	d, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

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
