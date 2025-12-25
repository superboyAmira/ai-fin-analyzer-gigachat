package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
	"go.uber.org/zap"
)

type OCRService struct {
	llmService *LLMService
	logger     *zap.Logger
}

// NewOCRService creates a new OCR service instance using GigaChat Vision API
func NewOCRService(llmService *LLMService, logger *zap.Logger) *OCRService {
	return &OCRService{
		llmService: llmService,
		logger:     logger,
	}
}

// ExtractText extracts text from an image or PDF file
// For PDF: uses go-fitz library for direct text extraction
// For images: uses GigaChat Vision API
// Supports Russian and English languages for financial documents
// Supported formats: .jpg, .jpeg, .png, .pdf
func (s *OCRService) ExtractText(ctx context.Context, filePath string) (string, error) {
	// Validate file format
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedFormats := []string{".jpg", ".jpeg", ".png", ".pdf"}
	isSupported := false
	for _, format := range supportedFormats {
		if ext == format {
			isSupported = true
			break
		}
	}
	if !isSupported {
		return "", fmt.Errorf("unsupported file format: %s (supported: jpg, jpeg, png, pdf)", ext)
	}

	var text string
	var err error

	// Use different methods for PDF and images
	if ext == ".pdf" {
		// Extract text from PDF using go-fitz library
		text, err = s.extractTextFromPDF(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to extract text from PDF: %w", err)
		}
	} else {
		// Use GigaChat Vision API for images
		text, err = s.llmService.ExtractTextFromImage(ctx, filePath)
		if err != nil {
			return "", fmt.Errorf("failed to extract text with GigaChat Vision: %w", err)
		}
	}

	// Clean up extracted text
	text = strings.TrimSpace(text)

	// Determine file type for logging
	fileType := "image"
	if ext == ".pdf" {
		fileType = "PDF"
	}

	s.logger.Info("OCR extraction completed",
		zap.String("file", filePath),
		zap.String("type", fileType),
		zap.String("method", s.getExtractionMethod(ext)),
		zap.Int("text_length", len(text)),
	)

	if text == "" {
		return "", fmt.Errorf("no text extracted from %s", fileType)
	}

	return text, nil
}

// extractTextFromPDF extracts text from PDF using go-fitz library
func (s *OCRService) extractTextFromPDF(pdfPath string) (string, error) {
	// Open PDF document
	doc, err := fitz.New(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var textBuilder strings.Builder

	// Extract text from all pages
	for i := 0; i < doc.NumPage(); i++ {
		pageText, err := doc.Text(i)
		if err != nil {
			s.logger.Warn("Failed to extract text from page",
				zap.Int("page", i+1),
				zap.String("file", pdfPath),
				zap.Error(err),
			)
			continue
		}

		if pageText != "" {
			textBuilder.WriteString(pageText)
			textBuilder.WriteString("\n") // Add newline between pages
		}
	}

	text := textBuilder.String()
	text = strings.TrimSpace(text)

	if text == "" {
		return "", fmt.Errorf("no text found in PDF")
	}

	s.logger.Info("PDF text extracted using go-fitz",
		zap.String("file", pdfPath),
		zap.Int("pages", doc.NumPage()),
		zap.Int("text_length", len(text)),
	)

	return text, nil
}

// getExtractionMethod returns the method name used for extraction
func (s *OCRService) getExtractionMethod(ext string) string {
	if ext == ".pdf" {
		return "go-fitz"
	}
	return "GigaChat Vision"
}

// ExtractTextFromReader extracts text from an image or PDF reader
func (s *OCRService) ExtractTextFromReader(ctx context.Context, reader io.Reader, format string) (string, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "ocr-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy reader to temp file
	if _, err := io.Copy(tmpFile, reader); err != nil {
		return "", fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add extension based on format
	ext := ".jpg"
	switch format {
	case "image/png":
		ext = ".png"
	case "application/pdf":
		ext = ".pdf"
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	}

	newPath := tmpFile.Name() + ext
	if err := os.Rename(tmpFile.Name(), newPath); err != nil {
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}
	defer os.Remove(newPath)

	return s.ExtractText(ctx, newPath)
}
