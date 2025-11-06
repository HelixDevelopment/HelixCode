package filesystem

import "fmt"

// FileSystemError represents a file system error
type FileSystemError struct {
	Type    ErrorType
	Path    string
	Message string
	Cause   error
}

func (e *FileSystemError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *FileSystemError) Unwrap() error {
	return e.Cause
}

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorFileNotFound     ErrorType = "file_not_found"
	ErrorPermissionDenied ErrorType = "permission_denied"
	ErrorInvalidPath      ErrorType = "invalid_path"
	ErrorFileExists       ErrorType = "file_exists"
	ErrorIsDirectory      ErrorType = "is_directory"
	ErrorNotDirectory     ErrorType = "not_directory"
	ErrorFileTooLarge     ErrorType = "file_too_large"
	ErrorInvalidEncoding  ErrorType = "invalid_encoding"
	ErrorDiskFull         ErrorType = "disk_full"
	ErrorTimeout          ErrorType = "timeout"
)

// SecurityError represents a security-related error
type SecurityError struct {
	Type    string
	Message string
	Path    string
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("security error [%s]: %s (path: %s)", e.Type, e.Message, e.Path)
}

// Common error constructors
func NewFileNotFoundError(path string) error {
	return &FileSystemError{
		Type:    ErrorFileNotFound,
		Path:    path,
		Message: "file not found",
	}
}

func NewPermissionDeniedError(path string, cause error) error {
	return &FileSystemError{
		Type:    ErrorPermissionDenied,
		Path:    path,
		Message: "permission denied",
		Cause:   cause,
	}
}

func NewFileTooLargeError(path string, size, limit int64) error {
	return &FileSystemError{
		Type:    ErrorFileTooLarge,
		Path:    path,
		Message: fmt.Sprintf("file too large: %d bytes (limit: %d bytes)", size, limit),
	}
}

func NewInvalidPathError(path string, cause error) error {
	return &FileSystemError{
		Type:    ErrorInvalidPath,
		Path:    path,
		Message: "invalid path",
		Cause:   cause,
	}
}

func NewFileExistsError(path string) error {
	return &FileSystemError{
		Type:    ErrorFileExists,
		Path:    path,
		Message: "file already exists",
	}
}

func NewIsDirectoryError(path string) error {
	return &FileSystemError{
		Type:    ErrorIsDirectory,
		Path:    path,
		Message: "path is a directory",
	}
}

func NewNotDirectoryError(path string) error {
	return &FileSystemError{
		Type:    ErrorNotDirectory,
		Path:    path,
		Message: "path is not a directory",
	}
}
