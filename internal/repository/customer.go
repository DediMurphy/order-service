package repository

import (
	"context"
	"database/sql"
	"errors"

	"order-service/internal/domain"
)

type CustomerRepository struct{}

func NewCustomerRepository() *CustomerRepository {
	return &CustomerRepository{}
}

func (r *CustomerRepository) Create(ctx context.Context, q Querier, c *domain.Customer) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO customers (name, email) VALUES (?, ?)`, c.Name, c.Email)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	got, err := r.FindByID(ctx, q, id)
	if err != nil {
		return err
	}
	*c = *got
	return nil
}

func (r *CustomerRepository) FindByID(ctx context.Context, q Querier, id int64) (*domain.Customer, error) {
	var c domain.Customer
	err := q.QueryRowContext(ctx,
		`SELECT id, name, email, created_at FROM customers WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Email, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
