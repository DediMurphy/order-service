package handler

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	r *gin.Engine,
	products *ProductHandler,
	customers *CustomerHandler,
	orders *OrderHandler,
) {
	r.POST("/products", products.Create)
	r.GET("/products", products.List)
	r.GET("/products/:id", products.GetByID)
	r.PUT("/products/:id", products.Update)

	r.POST("/customers", customers.Create)

	r.POST("/orders", orders.Create)
	r.GET("/orders", orders.List)
	r.GET("/orders/:id", orders.GetByID)
	r.PATCH("/orders/:id/status", orders.UpdateStatus)
}
