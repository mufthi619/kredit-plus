package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"kredit-plus/internal/entity"
	"strings"
	"time"
)

type customerService struct {
	repo   entity.CustomerRepository
	logger *zap.Logger
}

func NewCustomerService(repo entity.CustomerRepository, logger *zap.Logger) entity.CustomerService {
	return &customerService{
		repo:   repo,
		logger: logger,
	}
}

func (s *customerService) Create(ctx context.Context, req entity.CreateCustomerRequest) (*entity.CustomerResponse, error) {
	if errors := req.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	existingCustomer, err := s.repo.GetByNIK(ctx, req.NIK)
	if err != nil {
		s.logger.Error("failed to check existing customer", zap.Error(err))
		return nil, fmt.Errorf("failed to check existing customer: %w", err)
	}
	if existingCustomer != nil {
		return nil, fmt.Errorf("customer with NIK %s already exists", req.NIK)
	}

	customer := &entity.Customer{
		ID:         uuid.New(),
		NIK:        req.NIK,
		FullName:   req.FullName,
		LegalName:  req.LegalName,
		BirthPlace: req.BirthPlace,
		BirthDate:  req.BirthDate,
		Salary:     req.Salary,
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		s.logger.Error("failed to create customer",
			zap.Error(err),
			zap.String("nik", req.NIK),
		)
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return s.toResponse(customer), nil
}

func (s *customerService) GetByID(ctx context.Context, id uuid.UUID) (*entity.CustomerResponse, error) {
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get customer by ID",
			zap.Error(err),
			zap.String("customer_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	return s.toResponse(customer), nil
}

func (s *customerService) GetByNIK(ctx context.Context, nik string) (*entity.CustomerResponse, error) {
	if len(nik) != 16 {
		return nil, fmt.Errorf("invalid NIK format")
	}

	customer, err := s.repo.GetByNIK(ctx, nik)
	if err != nil {
		s.logger.Error("failed to get customer by NIK",
			zap.Error(err),
			zap.String("nik", nik),
		)
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	return s.toResponse(customer), nil
}

func (s *customerService) Update(ctx context.Context, id uuid.UUID, req entity.UpdateCustomerRequest) (*entity.CustomerResponse, error) {
	if errors := req.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get customer for update",
			zap.Error(err),
			zap.String("customer_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	if !customer.IsActive {
		return nil, fmt.Errorf("cannot update inactive customer")
	}

	customer.FullName = req.FullName
	customer.LegalName = req.LegalName
	customer.BirthPlace = req.BirthPlace
	customer.BirthDate = req.BirthDate
	customer.Salary = req.Salary
	customer.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, customer); err != nil {
		s.logger.Error("failed to update customer",
			zap.Error(err),
			zap.String("customer_id", id.String()),
		)
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	return s.toResponse(customer), nil
}

func (s *customerService) Delete(ctx context.Context, id uuid.UUID) error {
	customer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get customer for deletion",
			zap.Error(err),
			zap.String("customer_id", id.String()),
		)
		return fmt.Errorf("failed to get customer: %w", err)
	}

	if customer == nil {
		return fmt.Errorf("customer not found")
	}

	if !customer.IsActive {
		return fmt.Errorf("customer is already inactive")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete customer",
			zap.Error(err),
			zap.String("customer_id", id.String()),
		)
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	return nil
}

func (s *customerService) UploadDocument(ctx context.Context, customerID uuid.UUID, req entity.UploadDocumentRequest) (*entity.CustomerDocumentResponse, error) {
	if errors := req.Validate(); len(errors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	customer, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		s.logger.Error("failed to get customer for document upload",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	if !customer.IsActive {
		return nil, fmt.Errorf("cannot upload document for inactive customer")
	}

	filter := entity.DocumentFilterRepository{
		CustomerID:   customerID,
		DocumentType: &req.DocumentType,
		Limit:        1,
		Offset:       0,
	}

	existingDocs, _, err := s.repo.GetDocuments(ctx, filter)
	if err != nil {
		s.logger.Error("failed to check existing documents",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, fmt.Errorf("failed to check existing documents: %w", err)
	}

	if len(existingDocs) > 0 {
		return nil, fmt.Errorf("document type %s already exists for customer", req.DocumentType)
	}

	doc := &entity.CustomerDocument{
		ID:           uuid.New(),
		CustomerID:   customerID,
		DocumentType: req.DocumentType,
		DocumentURL:  req.DocumentURL,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreateDocument(ctx, doc); err != nil {
		s.logger.Error("failed to create customer document",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
			zap.String("document_type", string(req.DocumentType)),
		)
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return s.toDocumentResponse(doc), nil
}

func (s *customerService) GetDocuments(ctx context.Context, customerID uuid.UUID, filter entity.DocumentFilterRequest) ([]entity.CustomerDocumentResponse, int64, error) {
	if errors := filter.Validate(); len(errors) > 0 {
		return nil, 0, fmt.Errorf("validation failed: %v", strings.Join(errors, "||"))
	}

	customer, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		s.logger.Error("failed to get customer for documents",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, 0, fmt.Errorf("failed to get customer: %w", err)
	}

	if customer == nil {
		return nil, 0, fmt.Errorf("customer not found")
	}

	docs, count, err := s.repo.GetDocuments(ctx, filter.ToDocumentFilterRepo(customerID))
	if err != nil {
		s.logger.Error("failed to get customer documents",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return nil, 0, fmt.Errorf("failed to get documents: %w", err)
	}

	responses := make([]entity.CustomerDocumentResponse, len(docs))
	for i, doc := range docs {
		responses[i] = *s.toDocumentResponse(&doc)
	}

	return responses, count, nil
}

func (s *customerService) toResponse(customer *entity.Customer) *entity.CustomerResponse {
	response := &entity.CustomerResponse{
		ID:         customer.ID,
		NIK:        customer.NIK,
		FullName:   customer.FullName,
		LegalName:  customer.LegalName,
		BirthPlace: customer.BirthPlace,
		BirthDate:  customer.BirthDate.Format("2006-01-02"),
		Salary:     customer.Salary,
		IsActive:   customer.IsActive,
		CreatedAt:  customer.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  customer.UpdatedAt.Format(time.RFC3339),
	}

	if len(customer.Documents) > 0 {
		response.Documents = make([]entity.CustomerDocumentResponse, len(customer.Documents))
		for i, doc := range customer.Documents {
			response.Documents[i] = *s.toDocumentResponse(&doc)
		}
	}

	return response
}

func (s *customerService) toDocumentResponse(doc *entity.CustomerDocument) *entity.CustomerDocumentResponse {
	return &entity.CustomerDocumentResponse{
		ID:           doc.ID,
		CustomerID:   doc.CustomerID,
		DocumentType: doc.DocumentType,
		DocumentURL:  doc.DocumentURL,
		CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    doc.UpdatedAt.Format(time.RFC3339),
	}
}