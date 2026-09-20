package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"order-service/internal/domain"
	"order-service/internal/service"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ProductListResponse struct {
	Data []domain.Product `json:"data"`
	Meta Meta             `json:"meta"`
}

func (h *ProductHandler) Create(c *gin.Context) {
	var in service.CreateProductInput
	if !bindJSON(c, &in) {
		return
	}
	product, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	product, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("q")

	result, err := h.svc.List(c.Request.Context(), search, page, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ProductListResponse{
		Data: result.Products,
		Meta: Meta{
			Page: result.Page, Limit: result.Limit,
			Total: result.Total, TotalPages: result.TotalPages,
		},
	})
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var in service.UpdateProductInput
	if !bindJSON(c, &in) {
		return
	}
	product, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}
