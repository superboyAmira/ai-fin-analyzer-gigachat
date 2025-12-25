package handlers

import (
	"rag-iishka/internal/models"
	"rag-iishka/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DocumentHandler struct {
	docService *service.DocumentService
	logger     *zap.Logger
}

func NewDocumentHandler(docService *service.DocumentService, logger *zap.Logger) *DocumentHandler {
	return &DocumentHandler{
		docService: docService,
		logger:     logger,
	}
}

// UploadDocument godoc
// @Summary Upload a financial document
// @Description Upload a receipt, statement, or screenshot for processing
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file (image)"
// @Param type formData string true "Document type: receipt, statement, or screenshot"
// @Security Bearer
// @Success 201 {object} dto.DocumentResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/documents/upload [post]
func (h *DocumentHandler) UploadDocument(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File is required",
		})
	}

	docTypeStr := c.FormValue("type")
	if docTypeStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Type is required",
		})
	}

	var docType models.DocumentType
	switch docTypeStr {
	case "receipt":
		docType = models.DocumentTypeReceipt
	case "statement":
		docType = models.DocumentTypeStatement
	case "screenshot":
		docType = models.DocumentTypeScreenshot
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document type",
		})
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to open file",
		})
	}
	defer src.Close()

	// Upload document
	doc, err := h.docService.UploadDocument(c.Context(), userID, src, file.Filename, docType)
	if err != nil {
		h.logger.Error("Failed to upload document", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to upload document",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(doc)
}

// ProcessDocument godoc
// @Summary Process a document
// @Description Process a document: OCR -> LLM analysis -> RAG -> recommendations
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Security Bearer
// @Success 200 {object} dto.ProcessDocumentResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/documents/{id}/process [post]
func (h *DocumentHandler) ProcessDocument(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	documentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	result, err := h.docService.ProcessDocument(c.Context(), userID, documentID)
	if err != nil {
		h.logger.Error("Failed to process document", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process document",
		})
	}

	return c.JSON(result)
}

// ListDocuments godoc
// @Summary List user's documents
// @Description Get a list of user's uploaded documents
// @Tags documents
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Security Bearer
// @Success 200 {array} dto.DocumentResponse
// @Failure 401 {object} map[string]string
// @Router /api/v1/documents [get]
func (h *DocumentHandler) ListDocuments(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	limit := c.QueryInt("limit", 10)
	offset := c.QueryInt("offset", 0)

	docs, err := h.docService.ListDocuments(c.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list documents", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list documents",
		})
	}

	return c.JSON(docs)
}

func getUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userIDStr, ok := c.Locals("userID").(string)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

