package validator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"pot-play-storage/internal/model"
)

// File size limits
const (
	MaxFileSize     = 100 * 1024 * 1024 // 100MB
	MaxFilenameLen  = 255
	MinFilenameLen  = 1
)

// Allowed file types and their MIME types
var AllowedTypes = map[string][]string{
	".jpg":  {"image/jpeg"},
	".jpeg": {"image/jpeg"},
	".png":  {"image/png"},
	".gif":  {"image/gif"},
	".webp": {"image/webp"},
	".avif": {"image/avif"},
	".mp4":  {"video/mp4"},
	".avi":  {"video/x-msvideo"},
	".mov":  {"video/quicktime"},
	".pdf":  {"application/pdf"},
	".txt":  {"text/plain", "text/plain; charset=utf-8"},
	".zip":  {"application/zip"},
}

// Dangerous file extensions that should be blocked
var DangerousTypes = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".com": true,
	".scr": true, ".pif": true, ".vbs": true, ".js": true,
	".jar": true, ".sh": true, ".bash": true, ".ps1": true,
	".php": true, ".asp": true, ".jsp": true, ".py": true,
	".rb": true, ".pl": true, ".cgi": true, ".dll": true,
	".so": true, ".dylib": true, ".app": true, ".deb": true,
	".rpm": true, ".msi": true, ".dmg": true, ".iso": true,
}

// ValidateFilename checks if filename is safe and valid
func ValidateFilename(filename string) error {
	if len(filename) < MinFilenameLen {
		return fmt.Errorf("filename too short")
	}
	
	if len(filename) > MaxFilenameLen {
		return fmt.Errorf("filename too long (max %d characters)", MaxFilenameLen)
	}
	
	// Check for dangerous characters
	dangerousChars := []rune{'<', '>', ':', '"', '|', '?', '*', '\x00'}
	for _, char := range filename {
		for _, dangerous := range dangerousChars {
			if char == dangerous {
				return fmt.Errorf("filename contains invalid character: %c", char)
			}
		}
		
		// Check for control characters
		if unicode.IsControl(char) {
			return fmt.Errorf("filename contains control character")
		}
	}
	
	// Check for reserved Windows filenames
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	
	baseFilename := strings.ToUpper(strings.TrimSuffix(filename, filepath.Ext(filename)))
	for _, reserved := range reservedNames {
		if baseFilename == reserved {
			return fmt.Errorf("filename uses reserved name: %s", reserved)
		}
	}
	
	// Check for path traversal attempts
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("filename contains path traversal characters")
	}
	
	// Check for hidden files (starting with .)
	if strings.HasPrefix(filename, ".") {
		return fmt.Errorf("hidden files not allowed")
	}
	
	return nil
}

// ValidateFileExtension checks if file extension is allowed
func ValidateFileExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	
	if ext == "" {
		return fmt.Errorf("file must have an extension")
	}
	
	// Check if extension is dangerous
	if DangerousTypes[ext] {
		return fmt.Errorf("dangerous file type not allowed: %s", ext)
	}
	
	// Check if extension is in allowed list
	if _, allowed := AllowedTypes[ext]; !allowed {
		return fmt.Errorf("unsupported file type: %s", ext)
	}
	
	return nil
}

// ValidateContentType checks if MIME type matches the file extension
func ValidateContentType(filename, contentType string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	allowedMimeTypes, exists := AllowedTypes[ext]
	
	if !exists {
		return fmt.Errorf("file extension not allowed: %s", ext)
	}
	
	// Check if content type matches expected types for this extension
	for _, allowedType := range allowedMimeTypes {
		if contentType == allowedType {
			return nil
		}
	}
	
	return fmt.Errorf("content type %s does not match file extension %s", contentType, ext)
}

// ValidateFileSize checks if file size is within limits
func ValidateFileSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("file size must be greater than 0")
	}
	
	if size > MaxFileSize {
		return fmt.Errorf("file too large (max %d bytes)", MaxFileSize)
	}
	
	return nil
}

// ValidateFile performs comprehensive validation of file header
func ValidateFile(header *model.FileHeader) error {
	if header == nil {
		return fmt.Errorf("file header is required")
	}
	
	// Validate filename
	if err := ValidateFilename(header.Name); err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}
	
	// Validate file extension
	if err := ValidateFileExtension(header.Name); err != nil {
		return fmt.Errorf("invalid file extension: %w", err)
	}
	
	// Validate file size
	if err := ValidateFileSize(header.Size); err != nil {
		return fmt.Errorf("invalid file size: %w", err)
	}
	
	// Validate content type
	if header.ContentType != "" {
		if err := ValidateContentType(header.Name, header.ContentType); err != nil {
			return fmt.Errorf("invalid content type: %w", err)
		}
	}
	
	return nil
}

// SanitizeFilename removes or replaces dangerous characters in filename
func SanitizeFilename(filename string) string {
	// Replace dangerous characters with underscore
	reg := regexp.MustCompile(`[<>:"|?*\x00-\x1f]`)
	sanitized := reg.ReplaceAllString(filename, "_")
	
	// Remove leading/trailing spaces and dots
	sanitized = strings.Trim(sanitized, " .")
	
	// Limit length
	if len(sanitized) > MaxFilenameLen {
		ext := filepath.Ext(sanitized)
		nameOnly := strings.TrimSuffix(sanitized, ext)
		maxNameLen := MaxFilenameLen - len(ext)
		if maxNameLen > 0 {
			sanitized = nameOnly[:maxNameLen] + ext
		}
	}
	
	// Ensure we have a valid filename
	if sanitized == "" {
		sanitized = "unnamed_file"
	}
	
	return sanitized
}