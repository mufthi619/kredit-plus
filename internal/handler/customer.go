package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"kredit-plus/internal/entity"
	"kredit-plus/utils/response_formatter"
	"strconv"
)

type CustomerHandler struct {
	service entity.CustomerService
	logger  *zap.Logger
}

func NewCustomerHandler(service entity.CustomerService, logger *zap.Logger) *CustomerHandler {
	return &CustomerHandler{
		service: service,
		logger:  logger,
	}
}

func (h *CustomerHandler) RegisterRoutes(app *fiber.App) {
	customers := app.Group("/api/v1/customers")

	//Customer management
	customers.Post("", h.Create)
	customers.Get("/:id", h.GetByID)
	customers.Get("/nik/:nik", h.GetByNIK)
	customers.Put("/:id", h.Update)
	customers.Delete("/:id", h.Delete)

	//Document management
	customers.Post("/:id/documents", h.UploadDocument)
	customers.Get("/:id/documents", h.GetDocuments)
}

func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	var req entity.CreateCustomerRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("failed to parse create customer request", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	customer, err := h.service.Create(c.Context(), req)
	if err != nil {
		if err.Error() == "customer with NIK already exists" {
			return c.Status(fiber.StatusConflict).JSON(response_formatter.Error(
				fiber.StatusConflict,
				"Customer already exists",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to create customer", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to create customer",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusCreated).JSON(response_formatter.Created(
		customer,
		"Customer created successfully",
	))
}

func (h *CustomerHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid customer ID",
			[]string{err.Error()},
		))
	}

	customer, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err.Error() == "customer not found" {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Customer not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to get customer", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get customer",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		customer,
		"Customer retrieved successfully",
	))
}

func (h *CustomerHandler) GetByNIK(c *fiber.Ctx) error {
	nik := c.Params("nik")
	if len(nik) != 16 {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid NIK format",
			[]string{"NIK must be 16 characters"},
		))
	}

	customer, err := h.service.GetByNIK(c.Context(), nik)
	if err != nil {
		if err.Error() == "customer not found" {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Customer not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to get customer by NIK", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get customer",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		customer,
		"Customer retrieved successfully",
	))
}

func (h *CustomerHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid customer ID",
			[]string{err.Error()},
		))
	}

	var req entity.UpdateCustomerRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("failed to parse update customer request", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	customer, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		if err.Error() == "customer not found" {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Customer not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to update customer", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to update customer",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		customer,
		"Customer updated successfully",
	))
}

func (h *CustomerHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid customer ID",
			[]string{err.Error()},
		))
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		if err.Error() == "customer not found" {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Customer not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to delete customer", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to delete customer",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		nil,
		"Customer deleted successfully",
	))
}

func (h *CustomerHandler) UploadDocument(c *fiber.Ctx) error {
	customerID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid customer ID",
			[]string{err.Error()},
		))
	}

	var req entity.UploadDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("failed to parse upload document request", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	doc, err := h.service.UploadDocument(c.Context(), customerID, req)
	if err != nil {
		if err.Error() == "customer not found" {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Customer not found",
				[]string{err.Error()},
			))
		}

		if err.Error() == "document type already exists for customer" {
			return c.Status(fiber.StatusConflict).JSON(response_formatter.Error(
				fiber.StatusConflict,
				"Document already exists",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to upload document", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to upload document",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusCreated).JSON(response_formatter.Created(
		doc,
		"Document uploaded successfully",
	))
}

func (h *CustomerHandler) GetDocuments(c *fiber.Ctx) error {
	customerID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid customer ID",
			[]string{err.Error()},
		))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))
	page, perPage = response_formatter.ValidatePagination(page, perPage)

	var docType *entity.DocumentType
	if dt := c.Query("document_type"); dt != "" {
		t := entity.DocumentType(dt)
		if !t.IsValid() {
			return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
				fiber.StatusBadRequest,
				"Invalid document type",
				[]string{"document_type must be either 'ktp' or 'selfie'"},
			))
		}
		docType = &t
	}

	filter := entity.DocumentFilterRequest{
		DocumentType: docType,
		Page:         page,
		PerPage:      perPage,
	}

	documents, total, err := h.service.GetDocuments(c.Context(), customerID, filter)
	if err != nil {
		if err.Error() == "customer not found" {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Customer not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to get documents", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get documents",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.WithPagination(
		documents,
		"Documents retrieved successfully",
		page,
		perPage,
		total,
	))
}
