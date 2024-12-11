package entity

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"html"
	"time"
)

type (
	Asset struct {
		ID           uuid.UUID     `gorm:"type:char(36);primary_key"`
		Name         string        `gorm:"type:varchar(100);not null"`
		Category     string        `gorm:"type:varchar(50);not null"` //In Ex Case : (white_goods, motor, mobil)
		Description  string        `gorm:"type:text"`
		Price        float64       `gorm:"type:decimal(15,2);not null"`
		CreatedAt    time.Time     `gorm:"type:timestamp;not null"`
		UpdatedAt    time.Time     `gorm:"type:timestamp;not null"`
		Transactions []Transaction `gorm:"foreignKey:AssetID"`
	}

	AssetService interface {
		Create(ctx context.Context, req CreateAssetRequest) (*AssetResponse, error)
		GetByID(ctx context.Context, id uuid.UUID) (*AssetResponse, error)
		GetAll(ctx context.Context, filter AssetFilterRequest) ([]AssetResponse, int64, error)
		Update(ctx context.Context, id uuid.UUID, req UpdateAssetRequest) (*AssetResponse, error)
		Delete(ctx context.Context, id uuid.UUID) error
	}

	AssetRepository interface {
		Create(ctx context.Context, asset *Asset) error
		GetByID(ctx context.Context, id uuid.UUID) (*Asset, error)
		GetAllWithFilter(ctx context.Context, filter AssetFilterRepository) (assets []Asset, count int64, err error)
		Update(ctx context.Context, asset *Asset) error
		Delete(ctx context.Context, id uuid.UUID) error
	}

	AssetFilterRequest struct {
		Category string
		MinPrice float64
		MaxPrice float64
		Limit    int
		Offset   int
	}

	AssetFilterRepository struct {
		Category string
		MinPrice float64
		MaxPrice float64
		Limit    int
		Offset   int
	}

	CreateAssetRequest struct {
		Name        string  `json:"name" validate:"required,min=3,max=100"`
		Category    string  `json:"category" validate:"required,oneof=white_goods motor mobil"`
		Description string  `json:"description" validate:"required"`
		Price       float64 `json:"price" validate:"required,gt=0"`
	}

	UpdateAssetRequest struct {
		Name        string  `json:"name" validate:"required,min=3,max=100"`
		Description string  `json:"description" validate:"required"`
		Price       float64 `json:"price" validate:"required,gt=0"`
	}

	AssetResponse struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Category    string    `json:"category"`
		Description string    `json:"description"`
		Price       float64   `json:"price"`
		CreatedAt   string    `json:"created_at"`
		UpdatedAt   string    `json:"updated_at"`
	}
)

func (req CreateAssetRequest) Validate() []string {
	//XSS Protection
	html.EscapeString(req.Name)
	html.EscapeString(req.Description)
	html.EscapeString(req.Category)

	//Validate
	var errors []string
	err := validate.Struct(req)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Name":
				errors = append(errors, fmt.Sprintf("name %s", err.Tag()))
			case "Category":
				errors = append(errors, "category must be one of: white_goods, motor, mobil")
			case "Description":
				errors = append(errors, fmt.Sprintf("description %s", err.Tag()))
			case "Price":
				errors = append(errors, fmt.Sprintf("price must be greater than 0"))
			}
		}
	}
	return errors
}

func (req UpdateAssetRequest) Validate() []string {
	//XSS Protection
	html.EscapeString(req.Name)
	html.EscapeString(req.Description)

	//Validate
	var errors []string
	err := validate.Struct(req)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Name":
				errors = append(errors, fmt.Sprintf("name %s", err.Tag()))
			case "Description":
				errors = append(errors, fmt.Sprintf("description %s", err.Tag()))
			case "Price":
				errors = append(errors, fmt.Sprintf("price must be greater than 0"))
			}
		}
	}
	return errors
}

func (req AssetFilterRequest) ToAssetFilterRepo() AssetFilterRepository {
	return AssetFilterRepository{
		Category: req.Category,
		MinPrice: req.MinPrice,
		MaxPrice: req.MaxPrice,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}
}
