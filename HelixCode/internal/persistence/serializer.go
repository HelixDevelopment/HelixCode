package persistence

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
)

// Format represents a serialization format
type Format string

const (
	FormatJSON     Format = "json"    // JSON format
	FormatJSONGzip Format = "json.gz" // Compressed JSON
	FormatBinary   Format = "bin"     // Binary format (future)
)

// Serializer handles serialization and deserialization
type Serializer interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
	Format() Format
	Extension() string
}

// JSONSerializer serializes data as JSON
type JSONSerializer struct {
	indent bool
}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{
		indent: true,
	}
}

// NewCompactJSONSerializer creates a JSON serializer without indentation
func NewCompactJSONSerializer() *JSONSerializer {
	return &JSONSerializer{
		indent: false,
	}
}

// Serialize serializes data to JSON
func (s *JSONSerializer) Serialize(v interface{}) ([]byte, error) {
	if s.indent {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}

// Deserialize deserializes JSON data
func (s *JSONSerializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Format returns the format
func (s *JSONSerializer) Format() Format {
	return FormatJSON
}

// Extension returns the file extension
func (s *JSONSerializer) Extension() string {
	return ".json"
}

// JSONGzipSerializer serializes data as compressed JSON
type JSONGzipSerializer struct {
	indent bool
}

// NewJSONGzipSerializer creates a new compressed JSON serializer
func NewJSONGzipSerializer() *JSONGzipSerializer {
	return &JSONGzipSerializer{
		indent: true,
	}
}

// Serialize serializes data to compressed JSON
func (s *JSONGzipSerializer) Serialize(v interface{}) ([]byte, error) {
	// First serialize to JSON
	var jsonData []byte
	var err error
	if s.indent {
		jsonData, err = json.MarshalIndent(v, "", "  ")
	} else {
		jsonData, err = json.Marshal(v)
	}
	if err != nil {
		return nil, err
	}

	// Compress with gzip
	return compressGzip(jsonData)
}

// Deserialize deserializes compressed JSON data
func (s *JSONGzipSerializer) Deserialize(data []byte, v interface{}) error {
	// Decompress
	jsonData, err := decompressGzip(data)
	if err != nil {
		return err
	}

	// Deserialize JSON
	return json.Unmarshal(jsonData, v)
}

// Format returns the format
func (s *JSONGzipSerializer) Format() Format {
	return FormatJSONGzip
}

// Extension returns the file extension
func (s *JSONGzipSerializer) Extension() string {
	return ".json.gz"
}

// compressGzip compresses data with gzip
func compressGzip(data []byte) ([]byte, error) {
	var buf []byte
	writer := gzip.NewWriter(&byteSliceWriter{buf: &buf})

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

// decompressGzip decompresses gzip data
func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(&byteSliceReader{buf: data})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	result := make([]byte, 0)
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// byteSliceWriter wraps a byte slice for writing
type byteSliceWriter struct {
	buf *[]byte
}

func (w *byteSliceWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// byteSliceReader wraps a byte slice for reading
type byteSliceReader struct {
	buf []byte
	pos int
}

func (r *byteSliceReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.buf) {
		return 0, io.EOF
	}

	n = copy(p, r.buf[r.pos:])
	r.pos += n

	return n, nil
}

// Validate checks if data appears to be in the expected format
func Validate(data []byte, format Format) error {
	switch format {
	case FormatJSON:
		return validateJSON(data)
	case FormatJSONGzip:
		return validateJSONGzip(data)
	case FormatBinary:
		return validateBinary(data)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

// validateJSON validates JSON data
func validateJSON(data []byte) error {
	var temp interface{}
	return json.Unmarshal(data, &temp)
}

// validateJSONGzip validates compressed JSON data
func validateJSONGzip(data []byte) error {
	jsonData, err := decompressGzip(data)
	if err != nil {
		return err
	}
	return validateJSON(jsonData)
}

// validateBinary validates binary data
func validateBinary(data []byte) error {
	// Simple check: must have at least 4 bytes for header
	if len(data) < 4 {
		return fmt.Errorf("binary data too short")
	}
	return nil
}

// DetectFormat attempts to detect the format from data
func DetectFormat(data []byte) (Format, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty data")
	}

	// Check for gzip magic bytes (0x1f, 0x8b)
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return FormatJSONGzip, nil
	}

	// Check if it's valid JSON
	if err := validateJSON(data); err == nil {
		return FormatJSON, nil
	}

	// Default to binary
	return FormatBinary, nil
}
