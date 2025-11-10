package filesystem

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// FileReader provides methods for reading file contents
type FileReader interface {
	// Read reads the entire file content
	Read(ctx context.Context, path string) (*FileContent, error)

	// ReadLines reads specific lines from a file
	ReadLines(ctx context.Context, path string, start, end int) (*FileContent, error)

	// ReadWithLimit reads file with size limit
	ReadWithLimit(ctx context.Context, path string, maxBytes int64) (*FileContent, error)

	// GetInfo returns file metadata
	GetInfo(ctx context.Context, path string) (*FileInfo, error)

	// Exists checks if a file exists
	Exists(ctx context.Context, path string) (bool, error)
}

// FileContent represents the content and metadata of a file
type FileContent struct {
	Path        string
	Content     []byte
	Lines       []string
	TotalLines  int
	Size        int64
	ModTime     time.Time
	IsPartial   bool
	StartLine   int
	EndLine     int
	Encoding    string
	LineEndings LineEndingType
}

// FileInfo contains file metadata
type FileInfo struct {
	Path          string
	Name          string
	Size          int64
	Mode          os.FileMode
	ModTime       time.Time
	IsDir         bool
	IsSymlink     bool
	SymlinkTarget string
	MimeType      string
	Encoding      string
	Checksum      string // SHA-256
}

// LineEndingType represents the type of line endings
type LineEndingType int

const (
	LineEndingUnknown LineEndingType = iota
	LineEndingLF                     // Unix/Linux (\n)
	LineEndingCRLF                   // Windows (\r\n)
	LineEndingCR                     // Old Mac (\r)
	LineEndingMixed                  // Mixed line endings
)

func (l LineEndingType) String() string {
	switch l {
	case LineEndingLF:
		return "LF"
	case LineEndingCRLF:
		return "CRLF"
	case LineEndingCR:
		return "CR"
	case LineEndingMixed:
		return "Mixed"
	default:
		return "Unknown"
	}
}

// fileReader implements FileReader
type fileReader struct {
	fs *FileSystemTools
}

// Read reads the entire file content
func (r *fileReader) Read(ctx context.Context, path string) (*FileContent, error) {
	// Validate path
	validationResult, err := r.fs.pathValidator.Validate(path)
	if err != nil {
		return nil, err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check permissions
	if err := r.fs.permissionChecker.CheckPermission(normalizedPath, OpRead); err != nil {
		return nil, err
	}

	// Check cache
	if r.fs.cacheManager != nil {
		if cached, ok := r.fs.cacheManager.Get(normalizedPath); ok {
			return r.cacheEntryToFileContent(cached), nil
		}
	}

	// Get file info
	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewFileNotFoundError(normalizedPath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if directory
	if info.IsDir() {
		return nil, &FileSystemError{
			Type:    ErrorIsDirectory,
			Path:    normalizedPath,
			Message: "path is a directory",
		}
	}

	// Check size limit
	if r.fs.config.MaxFileSize > 0 && info.Size() > r.fs.config.MaxFileSize {
		return nil, NewFileTooLargeError(normalizedPath, info.Size(), r.fs.config.MaxFileSize)
	}

	// Read file
	content, err := os.ReadFile(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Parse content
	fileContent := r.parseContent(normalizedPath, content, info)

	// Cache the result
	if r.fs.cacheManager != nil {
		r.fs.cacheManager.Set(normalizedPath, content, info.ModTime())
	}

	return fileContent, nil
}

// ReadLines reads specific lines from a file
func (r *fileReader) ReadLines(ctx context.Context, path string, start, end int) (*FileContent, error) {
	// Validate path
	validationResult, err := r.fs.pathValidator.Validate(path)
	if err != nil {
		return nil, err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check permissions
	if err := r.fs.permissionChecker.CheckPermission(normalizedPath, OpRead); err != nil {
		return nil, err
	}

	// Get file info
	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewFileNotFoundError(normalizedPath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, &FileSystemError{
			Type:    ErrorIsDirectory,
			Path:    normalizedPath,
			Message: "path is a directory",
		}
	}

	// Open file
	file, err := os.Open(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read lines
	scanner := bufio.NewScanner(file)
	var lines []string
	var allLines []string
	lineNum := 1

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()
		allLines = append(allLines, line)

		if lineNum >= start && (end <= 0 || lineNum <= end) {
			lines = append(lines, line)
		}

		lineNum++

		// Stop early if we've read all requested lines
		if end > 0 && lineNum > end {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	totalLines := len(allLines)
	if end <= 0 {
		end = totalLines
	}

	// Reconstruct content from lines
	content := []byte(strings.Join(lines, "\n"))
	if len(lines) > 0 {
		content = append(content, '\n')
	}

	fileContent := &FileContent{
		Path:        normalizedPath,
		Content:     content,
		Lines:       lines,
		TotalLines:  totalLines,
		Size:        int64(len(content)),
		ModTime:     info.ModTime(),
		IsPartial:   true,
		StartLine:   start,
		EndLine:     end,
		Encoding:    "UTF-8",
		LineEndings: detectLineEndings(content),
	}

	return fileContent, nil
}

// ReadWithLimit reads file with size limit
func (r *fileReader) ReadWithLimit(ctx context.Context, path string, maxBytes int64) (*FileContent, error) {
	// Validate path
	validationResult, err := r.fs.pathValidator.Validate(path)
	if err != nil {
		return nil, err
	}
	normalizedPath := validationResult.NormalizedPath

	// Check permissions
	if err := r.fs.permissionChecker.CheckPermission(normalizedPath, OpRead); err != nil {
		return nil, err
	}

	// Get file info
	info, err := os.Stat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewFileNotFoundError(normalizedPath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, &FileSystemError{
			Type:    ErrorIsDirectory,
			Path:    normalizedPath,
			Message: "path is a directory",
		}
	}

	// Open file
	file, err := os.Open(normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read up to limit
	buffer := make([]byte, maxBytes)
	n, err := io.ReadFull(file, buffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := buffer[:n]
	isPartial := info.Size() > maxBytes

	fileContent := r.parseContent(normalizedPath, content, info)
	fileContent.IsPartial = isPartial

	return fileContent, nil
}

// GetInfo returns file metadata
func (r *fileReader) GetInfo(ctx context.Context, path string) (*FileInfo, error) {
	// Validate path
	validationResult, err := r.fs.pathValidator.Validate(path)
	if err != nil {
		return nil, err
	}
	normalizedPath := validationResult.NormalizedPath

	// Get file info
	info, err := os.Lstat(normalizedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewFileNotFoundError(normalizedPath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileInfo := &FileInfo{
		Path:      normalizedPath,
		Name:      info.Name(),
		Size:      info.Size(),
		Mode:      info.Mode(),
		ModTime:   info.ModTime(),
		IsDir:     info.IsDir(),
		IsSymlink: info.Mode()&os.ModeSymlink != 0,
		Encoding:  "UTF-8",
	}

	// Resolve symlink
	if fileInfo.IsSymlink {
		target, err := os.Readlink(normalizedPath)
		if err == nil {
			fileInfo.SymlinkTarget = target
		}
	}

	// Detect MIME type from extension
	fileInfo.MimeType = detectMimeType(normalizedPath)

	return fileInfo, nil
}

// Exists checks if a file exists
func (r *fileReader) Exists(ctx context.Context, path string) (bool, error) {
	// Validate path
	validationResult, err := r.fs.pathValidator.Validate(path)
	if err != nil {
		// If path is invalid due to security reasons, return false
		if _, ok := err.(*SecurityError); ok {
			return false, err
		}
		return false, err
	}

	_, err = os.Stat(validationResult.NormalizedPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// parseContent parses file content and extracts metadata
func (r *fileReader) parseContent(path string, content []byte, info os.FileInfo) *FileContent {
	// Split into lines
	lines := splitLines(content)

	// Detect line endings
	lineEndings := detectLineEndings(content)

	// Detect encoding
	encoding := detectEncoding(content)

	return &FileContent{
		Path:        path,
		Content:     content,
		Lines:       lines,
		TotalLines:  len(lines),
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		IsPartial:   false,
		StartLine:   1,
		EndLine:     len(lines),
		Encoding:    encoding,
		LineEndings: lineEndings,
	}
}

// cacheEntryToFileContent converts a cache entry to file content
func (r *fileReader) cacheEntryToFileContent(entry *CacheEntry) *FileContent {
	lines := splitLines(entry.Content)
	return &FileContent{
		Path:        entry.Path,
		Content:     entry.Content,
		Lines:       lines,
		TotalLines:  len(lines),
		Size:        entry.Size,
		ModTime:     entry.ModTime,
		IsPartial:   false,
		StartLine:   1,
		EndLine:     len(lines),
		Encoding:    "UTF-8",
		LineEndings: detectLineEndings(entry.Content),
	}
}

// splitLines splits content into lines preserving empty lines
func splitLines(content []byte) []string {
	if len(content) == 0 {
		return []string{}
	}

	// Use bufio.Scanner for efficient line splitting
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// detectLineEndings detects the type of line endings in content
func detectLineEndings(content []byte) LineEndingType {
	if len(content) == 0 {
		return LineEndingUnknown
	}

	hasCRLF := bytes.Contains(content, []byte("\r\n"))
	hasLF := bytes.Contains(content, []byte("\n"))
	hasCR := bytes.Contains(content, []byte("\r"))

	if hasCRLF && !hasCR {
		return LineEndingCRLF
	}
	if hasLF && !hasCR && !hasCRLF {
		return LineEndingLF
	}
	if hasCR && !hasLF {
		return LineEndingCR
	}
	if (hasCRLF && hasLF) || (hasCRLF && hasCR) || (hasLF && hasCR) {
		return LineEndingMixed
	}

	return LineEndingUnknown
}

// detectEncoding detects the character encoding of content
func detectEncoding(content []byte) string {
	if len(content) == 0 {
		return "UTF-8"
	}

	// Check for UTF-8 BOM
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		return "UTF-8-BOM"
	}

	// Check if valid UTF-8
	if utf8.Valid(content) {
		return "UTF-8"
	}

	// Check for common binary patterns
	if isBinary(content) {
		return "binary"
	}

	return "unknown"
}

// isBinary checks if content appears to be binary
func isBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 8KB for null bytes (common in binary files)
	checkLen := 8192
	if len(content) < checkLen {
		checkLen = len(content)
	}

	nullCount := 0
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			nullCount++
		}
	}

	// If more than 1% null bytes, likely binary
	return float64(nullCount)/float64(checkLen) > 0.01
}

// detectMimeType detects MIME type from file extension
func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".md":   "text/markdown",
		".go":   "text/x-go",
		".js":   "text/javascript",
		".ts":   "text/typescript",
		".jsx":  "text/jsx",
		".tsx":  "text/tsx",
		".py":   "text/x-python",
		".java": "text/x-java",
		".c":    "text/x-c",
		".cpp":  "text/x-c++",
		".h":    "text/x-c",
		".hpp":  "text/x-c++",
		".rs":   "text/x-rust",
		".json": "application/json",
		".xml":  "application/xml",
		".yaml": "application/x-yaml",
		".yml":  "application/x-yaml",
		".html": "text/html",
		".css":  "text/css",
		".sql":  "text/x-sql",
		".sh":   "text/x-sh",
		".bash": "text/x-sh",
		".zsh":  "text/x-sh",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}

	return "application/octet-stream"
}
