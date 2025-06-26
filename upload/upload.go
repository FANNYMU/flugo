package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flugo.com/config"
	"flugo.com/logger"
)

type UploadResult struct {
	FileName     string    `json:"file_name"`
	OriginalName string    `json:"original_name"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
	Path         string    `json:"path"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Extension    string    `json:"extension"`
	UploadedAt   time.Time `json:"uploaded_at"`
}

type UploadService struct {
	uploadPath    string
	maxFileSize   int64
	allowedTypes  []string
	enableResize  bool
	thumbnailSize int
}

func NewUploadService(cfg *config.UploadConfig) *UploadService {
	service := &UploadService{
		uploadPath:    cfg.UploadPath,
		maxFileSize:   cfg.MaxFileSize,
		allowedTypes:  cfg.AllowedTypes,
		enableResize:  cfg.EnableResize,
		thumbnailSize: cfg.ThumbnailSize,
	}

	if err := os.MkdirAll(cfg.UploadPath, 0755); err != nil {
		logger.Error("Failed to create upload directory: %v", err)
	}

	return service
}

var DefaultUploadService *UploadService

func Init(cfg *config.UploadConfig) {
	DefaultUploadService = NewUploadService(cfg)
}

func (u *UploadService) HandleUpload(r *http.Request, fieldName string) (*UploadResult, error) {
	if err := r.ParseMultipartForm(u.maxFileSize); err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	file, handler, err := r.FormFile(fieldName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from form: %w", err)
	}
	defer file.Close()

	if handler.Size > u.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", handler.Size, u.maxFileSize)
	}

	mimeType := handler.Header.Get("Content-Type")
	if !u.isAllowedType(mimeType) {
		return nil, fmt.Errorf("file type %s is not allowed", mimeType)
	}

	return u.saveFile(file, handler)
}

func (u *UploadService) HandleMultipleUploads(r *http.Request, fieldName string) ([]*UploadResult, error) {
	if err := r.ParseMultipartForm(u.maxFileSize); err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	files := r.MultipartForm.File[fieldName]
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found in field %s", fieldName)
	}

	var results []*UploadResult
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		if fileHeader.Size > u.maxFileSize {
			file.Close()
			continue
		}

		mimeType := fileHeader.Header.Get("Content-Type")
		if !u.isAllowedType(mimeType) {
			file.Close()
			continue
		}

		result, err := u.saveFile(file, fileHeader)
		file.Close()
		if err == nil {
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid files were uploaded")
	}

	return results, nil
}

func (u *UploadService) saveFile(file multipart.File, handler *multipart.FileHeader) (*UploadResult, error) {
	ext := filepath.Ext(handler.Filename)
	fileName := u.generateFileName(ext)
	filePath := filepath.Join(u.uploadPath, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	result := &UploadResult{
		FileName:     fileName,
		OriginalName: handler.Filename,
		Size:         size,
		MimeType:     handler.Header.Get("Content-Type"),
		Path:         filePath,
		URL:          "/uploads/" + fileName,
		Extension:    ext,
		UploadedAt:   time.Now(),
	}

	if u.enableResize && u.isImage(result.MimeType) {
		thumbnailName := u.generateThumbnailName(fileName)
		thumbnailPath := filepath.Join(u.uploadPath, thumbnailName)

		if err := u.createThumbnail(filePath, thumbnailPath); err == nil {
			result.ThumbnailURL = "/uploads/" + thumbnailName
		}
	}

	return result, nil
}

func (u *UploadService) generateFileName(ext string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%d%s", timestamp, ext)
}

func (u *UploadService) generateThumbnailName(fileName string) string {
	ext := filepath.Ext(fileName)
	name := strings.TrimSuffix(fileName, ext)
	return fmt.Sprintf("%s_thumb%s", name, ext)
}

func (u *UploadService) isAllowedType(mimeType string) bool {
	if len(u.allowedTypes) == 0 {
		return true
	}

	for _, allowedType := range u.allowedTypes {
		if allowedType == "*" || allowedType == mimeType {
			return true
		}

		if strings.HasSuffix(allowedType, "/*") {
			prefix := strings.TrimSuffix(allowedType, "/*")
			if strings.HasPrefix(mimeType, prefix+"/") {
				return true
			}
		}
	}

	return false
}

func (u *UploadService) isImage(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

func (u *UploadService) createThumbnail(srcPath, dstPath string) error {
	logger.Info("Creating thumbnail: %s -> %s", srcPath, dstPath)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (u *UploadService) DeleteFile(fileName string) error {
	filePath := filepath.Join(u.uploadPath, fileName)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	thumbnailName := u.generateThumbnailName(fileName)
	thumbnailPath := filepath.Join(u.uploadPath, thumbnailName)
	os.Remove(thumbnailPath)

	return nil
}

func (u *UploadService) GetFileInfo(fileName string) (*UploadResult, error) {
	filePath := filepath.Join(u.uploadPath, fileName)

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	ext := filepath.Ext(fileName)

	result := &UploadResult{
		FileName:   fileName,
		Size:       info.Size(),
		Path:       filePath,
		URL:        "/uploads/" + fileName,
		Extension:  ext,
		UploadedAt: info.ModTime(),
	}

	thumbnailName := u.generateThumbnailName(fileName)
	thumbnailPath := filepath.Join(u.uploadPath, thumbnailName)
	if _, err := os.Stat(thumbnailPath); err == nil {
		result.ThumbnailURL = "/uploads/" + thumbnailName
	}

	return result, nil
}

func (u *UploadService) ListFiles() ([]*UploadResult, error) {
	files, err := os.ReadDir(u.uploadPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload directory: %w", err)
	}

	var results []*UploadResult
	for _, file := range files {
		if file.IsDir() || strings.Contains(file.Name(), "_thumb") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		ext := filepath.Ext(file.Name())
		result := &UploadResult{
			FileName:   file.Name(),
			Size:       info.Size(),
			Path:       filepath.Join(u.uploadPath, file.Name()),
			URL:        "/uploads/" + file.Name(),
			Extension:  ext,
			UploadedAt: info.ModTime(),
		}

		thumbnailName := u.generateThumbnailName(file.Name())
		thumbnailPath := filepath.Join(u.uploadPath, thumbnailName)
		if _, err := os.Stat(thumbnailPath); err == nil {
			result.ThumbnailURL = "/uploads/" + thumbnailName
		}

		results = append(results, result)
	}

	return results, nil
}

func HandleUpload(r *http.Request, fieldName string) (*UploadResult, error) {
	if DefaultUploadService == nil {
		return nil, fmt.Errorf("upload service not initialized")
	}
	return DefaultUploadService.HandleUpload(r, fieldName)
}

func HandleMultipleUploads(r *http.Request, fieldName string) ([]*UploadResult, error) {
	if DefaultUploadService == nil {
		return nil, fmt.Errorf("upload service not initialized")
	}
	return DefaultUploadService.HandleMultipleUploads(r, fieldName)
}

func DeleteFile(fileName string) error {
	if DefaultUploadService == nil {
		return fmt.Errorf("upload service not initialized")
	}
	return DefaultUploadService.DeleteFile(fileName)
}

func GetFileInfo(fileName string) (*UploadResult, error) {
	if DefaultUploadService == nil {
		return nil, fmt.Errorf("upload service not initialized")
	}
	return DefaultUploadService.GetFileInfo(fileName)
}

func ListFiles() ([]*UploadResult, error) {
	if DefaultUploadService == nil {
		return nil, fmt.Errorf("upload service not initialized")
	}
	return DefaultUploadService.ListFiles()
}
