package service

import (
	"context"
	"database/sql"
	"fmt"
	"net/mail"
	"strings"

	"order-service/internal/domain"
	"order-service/internal/repository"
)

type CustomerService struct {
	db   *sql.DB
	repo *repository.CustomerRepository
}

func NewCustomerService(db *sql.DB, repo *repository.CustomerRepository) *CustomerService {
	return &CustomerService{db: db, repo: repo}
}

type CreateCustomerInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *CustomerService) Create(ctx context.Context, in CreateCustomerInput) (*domain.Customer, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.TrimSpace(in.Email)

	if in.Name == "" {
		return nil, domain.NewBadRequest("validation failed", "name is required")
	}
	
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return nil, domain.NewBadRequest("validation failed", "email is not a valid address")
	}

	c := &domain.Customer{Name: in.Name, Email: in.Email}
	if err := s.repo.Create(ctx, s.db, c); err != nil {
		if repository.IsUniqueViolation(err) {
			return nil, domain.NewConflict("duplicate email",
				fmt.Sprintf("customer with email '%s' already exists", in.Email))
		}
		return nil, err
	}
	return c, nil
}
