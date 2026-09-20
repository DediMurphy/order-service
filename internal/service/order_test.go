package service

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"order-service/internal/domain"
	"order-service/internal/repository"
)

func setup(t *testing.T) (*sql.DB, *OrderService, *ProductService, *CustomerService) {
	t.Helper()
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "test.db"))

	db, err := repository.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	productRepo := repository.NewProductRepository()
	customerRepo := repository.NewCustomerRepository()
	orderRepo := repository.NewOrderRepository()

	return db,
		NewOrderService(db, orderRepo, productRepo, customerRepo),
		NewProductService(db, productRepo),
		NewCustomerService(db, customerRepo)
}

func assertStatus(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with status %d, got nil", want)
	}
	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *domain.AppError, got %T: %v", err, err)
	}
	if appErr.Status != want {
		t.Fatalf("expected status %d, got %d (%s)", want, appErr.Status, appErr.Error())
	}
}

func seed(t *testing.T, ps *ProductService, cs *CustomerService) (*domain.Product, *domain.Customer) {
	t.Helper()
	ctx := context.Background()

	product, err := ps.Create(ctx, CreateProductInput{
		SKU: "KAOS-01", Name: "Kaos Polos", Price: 50000, Stock: 10,
	})
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}
	customer, err := cs.Create(ctx, CreateCustomerInput{
		Name: "Budi", Email: "budi@example.com",
	})
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	return product, customer
}

func currentStock(t *testing.T, db *sql.DB, productID int64) int {
	t.Helper()
	var stock int
	if err := db.QueryRow(`SELECT stock FROM products WHERE id = ?`, productID).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	return stock
}

func TestCreateOrder_Success(t *testing.T) {
	db, orderSvc, productSvc, customerSvc := setup(t)
	product, customer := seed(t, productSvc, customerSvc)

	order, err := orderSvc.Create(context.Background(), CreateOrderInput{
		CustomerID: customer.ID,
		Items:      []domain.OrderItemInput{{ProductID: product.ID, Qty: 3}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	if want := int64(150000); order.TotalAmount != want { // 50000 x 3
		t.Errorf("total_amount = %d, want %d", order.TotalAmount, want)
	}
	if order.Status != domain.StatusPending {
		t.Errorf("status = %s, want PENDING", order.Status)
	}
	if order.Items[0].PriceAtOrder != 50000 {
		t.Errorf("price_at_order = %d, want 50000", order.Items[0].PriceAtOrder)
	}
	if got := currentStock(t, db, product.ID); got != 7 {
		t.Errorf("stock = %d, want 7", got)
	}
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	_, orderSvc, productSvc, customerSvc := setup(t)
	product, customer := seed(t, productSvc, customerSvc)

	_, err := orderSvc.Create(context.Background(), CreateOrderInput{
		CustomerID: customer.ID,
		Items:      []domain.OrderItemInput{{ProductID: product.ID, Qty: 99}},
	})
	assertStatus(t, err, http.StatusBadRequest)
}

func TestCreateOrder_AtomicRollback(t *testing.T) {
	db, orderSvc, productSvc, customerSvc := setup(t)
	product, customer := seed(t, productSvc, customerSvc)

	second, err := productSvc.Create(context.Background(), CreateProductInput{
		SKU: "TOPI-01", Name: "Topi", Price: 25000, Stock: 1,
	})
	if err != nil {
		t.Fatalf("seed second product: %v", err)
	}

	_, err = orderSvc.Create(context.Background(), CreateOrderInput{
		CustomerID: customer.ID,
		Items: []domain.OrderItemInput{
			{ProductID: product.ID, Qty: 2}, 
			{ProductID: second.ID, Qty: 5},  
		},
	})
	assertStatus(t, err, http.StatusBadRequest)


	if got := currentStock(t, db, product.ID); got != 10 {
		t.Errorf("stock produk pertama = %d, want 10 (rollback gagal)", got)
	}
	if got := currentStock(t, db, second.ID); got != 1 {
		t.Errorf("stock produk kedua = %d, want 1", got)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if count != 0 {
		t.Errorf("jumlah order = %d, want 0 (tidak boleh ada order tersimpan)", count)
	}
}

func TestUpdateStatus_InvalidTransition(t *testing.T) {
	_, orderSvc, productSvc, customerSvc := setup(t)
	product, customer := seed(t, productSvc, customerSvc)
	ctx := context.Background()

	order, err := orderSvc.Create(ctx, CreateOrderInput{
		CustomerID: customer.ID,
		Items:      []domain.OrderItemInput{{ProductID: product.ID, Qty: 1}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	_, err = orderSvc.UpdateStatus(ctx, order.ID, domain.StatusShipped)
	assertStatus(t, err, http.StatusConflict)

	if _, err := orderSvc.UpdateStatus(ctx, order.ID, domain.StatusPaid); err != nil {
		t.Fatalf("PENDING -> PAID seharusnya boleh: %v", err)
	}

	_, err = orderSvc.UpdateStatus(ctx, order.ID, domain.StatusPending)
	assertStatus(t, err, http.StatusConflict)
}

func TestNotFound(t *testing.T) {
	_, orderSvc, productSvc, customerSvc := setup(t)
	_, customer := seed(t, productSvc, customerSvc)
	ctx := context.Background()

	_, err := productSvc.GetByID(ctx, 9999)
	assertStatus(t, err, http.StatusNotFound)

	_, err = orderSvc.GetByID(ctx, 9999)
	assertStatus(t, err, http.StatusNotFound)

	_, err = orderSvc.Create(ctx, CreateOrderInput{
		CustomerID: customer.ID,
		Items:      []domain.OrderItemInput{{ProductID: 9999, Qty: 1}},
	})
	assertStatus(t, err, http.StatusNotFound)
}

func TestUpdateStatus_CancelRestoresStock(t *testing.T) {
	db, orderSvc, productSvc, customerSvc := setup(t)
	product, customer := seed(t, productSvc, customerSvc)
	ctx := context.Background()

	order, err := orderSvc.Create(ctx, CreateOrderInput{
		CustomerID: customer.ID,
		Items:      []domain.OrderItemInput{{ProductID: product.ID, Qty: 4}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if got := currentStock(t, db, product.ID); got != 6 {
		t.Fatalf("stock setelah order = %d, want 6", got)
	}

	if _, err := orderSvc.UpdateStatus(ctx, order.ID, domain.StatusCancelled); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got := currentStock(t, db, product.ID); got != 10 {
		t.Errorf("stock setelah cancel = %d, want 10", got)
	}
}