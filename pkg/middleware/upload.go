package middleware

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	uploadDir    = "./uploads"
	maxFileSize  = 5 * 1024 * 1024 // 5MB
	allowedTypes = ".jpg,.jpeg,.png,.gif"
)

func FileUploadMiddleware() fiber.Handler {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create upload directory: %v", err))
	}

	return func(c *fiber.Ctx) error {
		// Get file from request
		file, err := c.FormFile("image")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No file uploaded",
			})
		}

		// Validate file size
		if file.Size > maxFileSize {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("File size exceeds maximum limit of %d bytes", maxFileSize),
			})
		}

		// Validate file type
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !strings.Contains(allowedTypes, ext) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("File type %s is not allowed. Allowed types: %s", ext, allowedTypes),
			})
		}

		// Generate unique filename
		filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
		filepath := filepath.Join(uploadDir, filename)

		// Save file
		if err := c.SaveFile(file, filepath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save file",
			})
		}

		// Store file info in context
		c.Locals("uploaded_file", map[string]string{
			"filename": filename,
			"path":     filepath,
			"url":      fmt.Sprintf("/uploads/%s", filename),
		})

		return c.Next()
	}
}

func MultipleFileUploadMiddleware() fiber.Handler {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create upload directory: %v", err))
	}

	return func(c *fiber.Ctx) error {
		// Parse multipart form
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No files uploaded",
			})
		}

		files := form.File["images"]
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No files uploaded",
			})
		}

		uploadedFiles := make([]map[string]string, 0)

		// Process each file
		for _, file := range files {
			if err := validateAndSaveFile(file, &uploadedFiles); err != nil {
				// Clean up already uploaded files on error
				for _, f := range uploadedFiles {
					os.Remove(f["path"])
				}
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": err.Error(),
				})
			}
		}

		// Store files info in context
		c.Locals("uploaded_files", uploadedFiles)

		return c.Next()
	}
}

func validateAndSaveFile(file *multipart.FileHeader, uploadedFiles *[]map[string]string) error {
	// Validate file size
	if file.Size > maxFileSize {
		return fmt.Errorf("file %s exceeds maximum size limit of %d bytes", file.Filename, maxFileSize)
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !strings.Contains(allowedTypes, ext) {
		return fmt.Errorf("file type %s is not allowed for %s. Allowed types: %s", ext, file.Filename, allowedTypes)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filepath := filepath.Join(uploadDir, filename)

	// Save file
	if err := saveFile(file, filepath); err != nil {
		return fmt.Errorf("failed to save file %s: %v", file.Filename, err)
	}

	// Add file info to uploaded files
	*uploadedFiles = append(*uploadedFiles, map[string]string{
		"filename": filename,
		"path":     filepath,
		"url":      fmt.Sprintf("/uploads/%s", filename),
	})

	return nil
}

func saveFile(file *multipart.FileHeader, filepath string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer dst.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := src.Read(buffer)
		if err != nil {
			break
		}
		dst.Write(buffer[:n])
	}

	return nil
}
