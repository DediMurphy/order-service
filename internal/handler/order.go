package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"order-service/internal/domain"
	"order-service/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) Create(c *gin.Context) {
	var in service.CreateOrderInput
	if !bindJSON(c, &in) {
		return
	}
	order, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetByID(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	order, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) List(c *gin.Context) {
	var customerID *int64
	if raw := c.Query("customer_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "validation failed", Detail: "customer_id must be a number"})
			return
		}
		customerID = &parsed
	}

	var status *string
	if raw := c.Query("status"); raw != "" {
		status = &raw
	}

	orders, err := h.svc.List(c.Request.Context(), customerID, status)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req updateStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	order, err := h.svc.UpdateStatus(c.Request.Context(), id, domain.OrderStatus(req.Status))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, order)
}
