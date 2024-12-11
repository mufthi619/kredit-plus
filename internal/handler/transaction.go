package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"kredit-plus/internal/entity"
	"kredit-plus/utils/response_formatter"
	"strconv"
)

type TransactionHandler struct {
	service entity.TransactionService
	logger  *zap.Logger
}

func NewTransactionHandler(service entity.TransactionService, logger *zap.Logger) *TransactionHandler {
	return &TransactionHandler{
		service: service,
		logger:  logger,
	}
}

func (h *TransactionHandler) RegisterRoutes(app *fiber.App) {
	transactions := app.Group("/api/v1/transactions")
	transactions.Post("", h.Create)
	transactions.Get("/:id", h.GetByID)
	transactions.Get("/contract/:contract_number", h.GetByContractNumber)
	transactions.Get("/customer/:customer_id", h.GetAllByCustomerID)
	transactions.Put("/:id/status", h.UpdateStatus)
}

func (h *TransactionHandler) Create(c *fiber.Ctx) error {
	var req entity.CreateTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("failed to parse create transaction request",
			zap.Error(err),
		)
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	transaction, err := h.service.Create(c.Context(), req)
	if err != nil {
		switch err {
		case entity.ErrDuplicateContract:
			return c.Status(fiber.StatusConflict).JSON(response_formatter.Error(
				fiber.StatusConflict,
				"Contract number already exists",
				[]string{err.Error()},
			))
		case entity.ErrInsufficientCreditLimit:
			return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
				fiber.StatusBadRequest,
				"Insufficient credit limit",
				[]string{err.Error()},
			))
		default:
			h.logger.Error("failed to create transaction",
				zap.Error(err),
				zap.Any("request", req),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
				fiber.StatusInternalServerError,
				"Failed to create transaction",
				[]string{err.Error()},
			))
		}
	}

	return c.Status(fiber.StatusCreated).JSON(response_formatter.Created(
		transaction,
		"Transaction created successfully",
	))
}

func (h *TransactionHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid transaction ID",
			[]string{err.Error()},
		))
	}

	transaction, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if err == entity.ErrTransactionNotFound {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Transaction not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to get transaction",
			zap.Error(err),
			zap.String("transaction_id", id.String()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get transaction",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		transaction,
		"Transaction retrieved successfully",
	))
}

func (h *TransactionHandler) GetByContractNumber(c *fiber.Ctx) error {
	contractNumber := c.Params("contract_number")
	if contractNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Contract number is required",
			[]string{"contract_number is required"},
		))
	}

	transaction, err := h.service.GetByContractNumber(c.Context(), contractNumber)
	if err != nil {
		if err == entity.ErrTransactionNotFound {
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Transaction not found",
				[]string{err.Error()},
			))
		}

		h.logger.Error("failed to get transaction by contract number",
			zap.Error(err),
			zap.String("contract_number", contractNumber),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get transaction",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		transaction,
		"Transaction retrieved successfully",
	))
}

func (h *TransactionHandler) GetAllByCustomerID(c *fiber.Ctx) error {
	customerID, err := uuid.Parse(c.Params("customer_id"))
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

	filter := entity.TransactionFilterRequest{
		Status:  entity.TransactionStatus(c.Query("status")),
		Page:    page,
		PerPage: perPage,
	}

	transactions, total, err := h.service.GetAllByCustomerID(c.Context(), customerID, filter)
	if err != nil {
		h.logger.Error("failed to get customer transactions",
			zap.Error(err),
			zap.String("customer_id", customerID.String()),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
			fiber.StatusInternalServerError,
			"Failed to get transactions",
			[]string{err.Error()},
		))
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.WithPagination(
		transactions,
		"Transactions retrieved successfully",
		page,
		perPage,
		total,
	))
}

type UpdateTransactionStatusRequest struct {
	Status entity.TransactionStatus `json:"status" validate:"required"`
}

func (h *TransactionHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid transaction ID",
			[]string{err.Error()},
		))
	}

	var req UpdateTransactionStatusRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("failed to parse update status request",
			zap.Error(err),
		)
		return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
			fiber.StatusBadRequest,
			"Invalid request body",
			[]string{err.Error()},
		))
	}

	if err := h.service.UpdateStatus(c.Context(), id, req.Status); err != nil {
		switch err {
		case entity.ErrTransactionNotFound:
			return c.Status(fiber.StatusNotFound).JSON(response_formatter.Error(
				fiber.StatusNotFound,
				"Transaction not found",
				[]string{err.Error()},
			))
		case entity.ErrInvalidStatus:
			return c.Status(fiber.StatusBadRequest).JSON(response_formatter.Error(
				fiber.StatusBadRequest,
				"Invalid status",
				[]string{err.Error()},
			))
		default:
			h.logger.Error("failed to update transaction status",
				zap.Error(err),
				zap.String("transaction_id", id.String()),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(response_formatter.Error(
				fiber.StatusInternalServerError,
				"Failed to update transaction status",
				[]string{err.Error()},
			))
		}
	}

	return c.Status(fiber.StatusOK).JSON(response_formatter.Success(
		nil,
		"Transaction status updated successfully",
	))
}
