package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"order-service/internal/domain"
	"order-service/internal/repository"
)

type ProductService struct {
	db    *sql.DB
	repo  *repository.ProductRepository
}

func NewProductService(db *sql.DB, repo *repository.ProductRepository) *ProductService {
	return &ProductService{db: db, repo: repo}
}

type CreateProductInput struct {
	SKU   string `json:"sku"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
	Stock int    `json:"stock"`
}

func (s *ProductService) Create(ctx context.Context, in CreateProductInput) (*domain.Product, error) {
	in.SKU = strings.TrimSpace(in.SKU)
	in.Name = strings.TrimSpace(in.Name)

	if in.SKU == "" {
		return nil, domain.NewBadRequest("validation failed", "sku is required")
	}
	if in.Name == "" {
		return nil, domain.NewBadRequest("validation failed", "name is required")
	}
	if in.Price <= 0 {
		return nil, domain.NewBadRequest("validation failed", "price must be greater than 0")
	}
	if in.Stock < 0 {
		return nil, domain.NewBadRequest("validation failed", "stock must be 0 or greater")
	}

	p := &domain.Product{SKU: in.SKU, Name: in.Name, Price: in.Price, Stock: in.Stock}
	if err := s.repo.Create(ctx, s.db, p); err != nil {		
		if repository.IsUniqueViolation(err) {
			return nil, domain.NewConflict("duplicate sku",
				fmt.Sprintf("product with sku '%s' already exists", in.SKU))
		}
		return nil, err
	}
	return p, nil
}

func (s *ProductService) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	p, err := s.repo.FindByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.NewNotFound(fmt.Sprintf("product with id %d not found", id))
	}
	return p, nil
}

type ListProductsResult struct {
	Products   []domain.Product
	Total      int
	Page       int
	Limit      int
	TotalPages int
}

func (s *ProductService) List(ctx context.Context, search string, page, limit int) (*ListProductsResult, error) {	
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	products, err := s.repo.List(ctx, s.db, search, limit, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.Count(ctx, s.db, search)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &ListProductsResult{
		Products: products, Total: total,
		Page: page, Limit: limit, TotalPages: totalPages,
	}, nil
}

type UpdateProductInput struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
	Stock int    `json:"stock"`
}

func (s *ProductService) Update(ctx context.Context, id int64, in UpdateProductInput) (*domain.Product, error) {
	p, err := s.repo.FindByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.NewNotFound(fmt.Sprintf("product with id %d not found", id))
	}

	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, domain.NewBadRequest("validation failed", "name is required")
	}
	if in.Price <= 0 {
		return nil, domain.NewBadRequest("validation failed", "price must be greater than 0")
	}
	if in.Stock < 0 {
		return nil, domain.NewBadRequest("validation failed", "stock must be 0 or greater")
	}

	p.Name, p.Price, p.Stock = in.Name, in.Price, in.Stock
	if err := s.repo.Update(ctx, s.db, p); err != nil {
		return nil, err
	}
	return p, nil
}
