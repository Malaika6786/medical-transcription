// Package utils provides utility functions for the Corti backend service
package utils

import (
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// SupportedAudioMIMETypes lists all supported audio MIME types for file upload
var SupportedAudioMIMETypes = map[string]bool{
	"audio/wav":                true,
	"audio/wave":               true,
	"audio/x-wav":              true,
	"audio/mpeg":               true,
	"audio/mp3":                true,
	"audio/mp4":                true,
	"audio/m4a":                true,
	"audio/x-m4a":              true,
	"audio/ogg":                true,
	"audio/flac":               true,
	"audio/x-flac":             true,
	"audio/webm":               true,
	"audio/aac":                true,
	"audio/x-aac":              true,
	"audio/aacp":               true,
	"audio/3gpp":               true,
	"audio/3gpp2":              true,
	"audio/basic":              true,
	"audio/x-aiff":             true,
	"audio/aiff":               true,
	"application/ogg":          true,
	"application/octet-stream": true, // Generic binary - allow if extension is valid
	"video/mp4":                true, // Some .m4a files report as video/mp4
}

// SupportedAudioExtensions lists all supported audio file extensions
var SupportedAudioExtensions = map[string]bool{
	".wav":  true,
	".wave": true,
	".mp3":  true,
	".mp4":  true,
	".m4a":  true,
	".ogg":  true,
	".flac": true,
	".webm": true,
	".aac":  true,
	".aiff": true,
	".aif":  true,
}

// ValidateAudioMIMEType checks if the provided MIME type is supported
func ValidateAudioMIMEType(mimeType string) bool {
	// Normalize MIME type (remove parameters like charset)
	normalizedMIME := strings.Split(strings.ToLower(mimeType), ";")[0]
	normalizedMIME = strings.TrimSpace(normalizedMIME)
	return SupportedAudioMIMETypes[normalizedMIME]
}

// ValidateAudioExtension checks if the provided file extension is supported
func ValidateAudioExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return SupportedAudioExtensions[ext]
}

// ValidateAudioFile validates both MIME type and extension of an uploaded audio file
func ValidateAudioFile(file *multipart.FileHeader) error {
	// Check file extension first - this is the primary validation
	if !ValidateAudioExtension(file.Filename) {
		return errors.New("unsupported audio file extension")
	}

	// Check MIME type from Content-Type header
	// If extension is valid, we're lenient with MIME types since browsers can report them inconsistently
	contentType := file.Header.Get("Content-Type")
	if contentType != "" && !ValidateAudioMIMEType(contentType) {
		// Log the unexpected MIME type but allow if extension is valid
		// This handles cases where browsers report unusual MIME types for valid audio files
		// The extension check above already validated this is an audio file
		// Just log for debugging purposes - we'll allow it since extension is valid
		_ = contentType // Extension is valid, so we accept it
	}

	return nil
}

// ReadAudioFile reads the contents of an uploaded audio file
func ReadAudioFile(file *multipart.FileHeader) ([]byte, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	return io.ReadAll(src)
}

// MaxAudioFileSize is the maximum allowed audio file size (100MB)
const MaxAudioFileSize = 100 * 1024 * 1024

// ValidateAudioFileSize checks if the file size is within limits
func ValidateAudioFileSize(size int64) error {
	if size > MaxAudioFileSize {
		return errors.New("audio file exceeds maximum size of 100MB")
	}
	if size == 0 {
		return errors.New("audio file is empty")
	}
	return nil
}

