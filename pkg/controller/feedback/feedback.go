package feedback

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/service"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

const (
	MaxTotalSize = 15 * 1024 * 1024 // 15 MB total
	MaxFileSize  = 8 * 1024 * 1024  // 8 MB per file
	MaxFiles     = 10               // Maximum number of files
)

var AllowedFileTypes = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".pdf":  true,
	".txt":  true,
	".md":   true,
}

type FeedbackController struct {
	feedbackService *service.FeedbackService
	userRepo        *repository.UserRepository
	logger          *utils.ErrorLogger
}

func NewFeedbackController() *FeedbackController {
	return &FeedbackController{
		feedbackService: service.NewFeedbackService(),
		userRepo:        repository.NewUserRepository(),
		logger:          utils.NewErrorLogger("feedback_controller"),
	}
}

// SendFeedback handles feedback submission with file attachments
// @Summary Send user feedback
// @Description Accepts user feedback with optional file attachments and sends to admin emails
// @Tags Feedback
// @Accept multipart/form-data
// @Produce json
// @Param message formData string true "Feedback message"
// @Param attachments formData file false "File attachments (max 15MB total, 8MB per file)"
// @Success 200 {object} map[string]interface{} "Feedback sent successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 413 {object} map[string]string "File too large"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/feedback/send [post]
func (c *FeedbackController) SendFeedback(w http.ResponseWriter, r *http.Request) {
	// Get user from session
	sessionUser := session.GetSessionExists(r)
	if sessionUser == nil {
		c.logger.LogWarning("User not authenticated", map[string]interface{}{
			"request_ip": r.RemoteAddr,
		})
		utils.SendErrorResponse(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Fetch user email from database
	dbUser, err := c.userRepo.GetByProviderUserID(r.Context(), sessionUser.Username)
	if err != nil {
		c.logger.LogError(err, "Failed to fetch user from database", map[string]interface{}{
			"username": sessionUser.Username,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch user information")
		return
	}

	// Get email from database
	userEmail := dbUser.Email.String
	if userEmail == "" {
		utils.SendErrorResponse(w, http.StatusNotFound, "User email not found")
		return
	}

	// Parse multipart form with max memory of 32MB
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		c.logger.LogWarning("Failed to parse multipart form", map[string]interface{}{
			"error": err.Error(),
		})
		utils.SendErrorResponse(w, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	// Extract message from form
	message := r.FormValue("message")

	// Validate required fields
	if message == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "Message is required")
		return
	}

	// Process file attachments
	var attachments []service.FileAttachment
	var totalSize int64

	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		files := r.MultipartForm.File["attachments"]

		if len(files) > MaxFiles {
			utils.SendErrorResponse(w, http.StatusBadRequest,
				fmt.Sprintf("Too many files. Maximum %d files allowed", MaxFiles))
			return
		}

		for _, fileHeader := range files {
			// Check individual file size
			if fileHeader.Size > MaxFileSize {
				utils.SendErrorResponse(w, http.StatusRequestEntityTooLarge,
					fmt.Sprintf("File '%s' exceeds maximum size of %d MB",
						fileHeader.Filename, MaxFileSize/(1024*1024)))
				return
			}

			// Check file extension
			ext := filepath.Ext(fileHeader.Filename)
			if !AllowedFileTypes[ext] {
				utils.SendErrorResponse(w, http.StatusBadRequest,
					fmt.Sprintf("File type '%s' is not allowed", ext))
				return
			}

			totalSize += fileHeader.Size

			// Check total size
			if totalSize > MaxTotalSize {
				utils.SendErrorResponse(w, http.StatusRequestEntityTooLarge,
					fmt.Sprintf("Total attachments exceed maximum size of %d MB",
						MaxTotalSize/(1024*1024)))
				return
			}

			// Open and read file
			file, err := fileHeader.Open()
			if err != nil {
				c.logger.LogError(err, "Failed to open uploaded file", map[string]interface{}{
					"filename": fileHeader.Filename,
				})
				utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to process uploaded file")
				return
			}
			defer file.Close()

			// Read file content
			content, err := io.ReadAll(file)
			if err != nil {
				c.logger.LogError(err, "Failed to read uploaded file", map[string]interface{}{
					"filename": fileHeader.Filename,
				})
				utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to read uploaded file")
				return
			}

			attachments = append(attachments, service.FileAttachment{
				Filename:    fileHeader.Filename,
				Content:     content,
				ContentType: fileHeader.Header.Get("Content-Type"),
				Size:        fileHeader.Size,
			})
		}
	}

	// Create feedback request
	feedbackReq := &service.SendFeedbackRequest{
		UserEmail:   userEmail,
		Message:     message,
		Attachments: attachments,
	}

	// Send feedback via service
	if err := c.feedbackService.SendFeedback(r.Context(), feedbackReq); err != nil {
		c.logger.LogError(err, "Failed to send feedback", map[string]interface{}{
			"user_email": userEmail,
		})
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to send feedback")
		return
	}

	c.logger.LogInfo("Feedback sent successfully", map[string]interface{}{
		"user_email":        userEmail,
		"attachments_count": len(attachments),
	})

	utils.SendSuccessResponseWithNoData(w, "Feedback sent successfully")
}
