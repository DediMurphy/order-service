package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"order-service/internal/service"
)

type CustomerHandler struct {
	svc *service.CustomerService
}

func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func (h *CustomerHandler) Create(c *gin.Context) {
	var in service.CreateCustomerInput
	if !bindJSON(c, &in) {
		return
	}
	customer, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, customer)
}
