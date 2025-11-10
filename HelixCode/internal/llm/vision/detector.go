package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectionMethod defines how to detect images
type DetectionMethod string

const (
	// DetectByMIME checks MIME type
	DetectByMIME DetectionMethod = "mime"

	// DetectByExtension checks file extension
	DetectByExtension DetectionMethod = "extension"

	// DetectByBase64 detects base64 encoded images
	DetectByBase64 DetectionMethod = "base64"

	// DetectByContent performs deep content inspection
	DetectByContent DetectionMethod = "content"

	// DetectByURL detects image URLs
	DetectByURL DetectionMethod = "url"
)

// ImageDetector detects images in user input
type ImageDetector struct {
	methods          []DetectionMethod
	contentInspector *ContentInspector
	config           *DetectionConfig
}

// DetectionResult contains detection results
type DetectionResult struct {
	HasImages       bool
	ImageCount      int
	Images          []*DetectedImage
	DetectionMethod DetectionMethod
	Confidence      float64
}

// DetectedImage represents a detected image
type DetectedImage struct {
	Source     ImageSource
	Location   string
	Format     string
	Size       int64
	Dimensions *Dimensions
	MIMEType   string
	Valid      bool
}

// ImageSource indicates where the image came from
type ImageSource string

const (
	SourceFile      ImageSource = "file"
	SourceURL       ImageSource = "url"
	SourceBase64    ImageSource = "base64"
	SourceClipboard ImageSource = "clipboard"
)

// Dimensions represents image dimensions
type Dimensions struct {
	Width  int
	Height int
}

// Input represents user input to be processed
type Input struct {
	Text        string
	Files       []*File
	Attachments []*Attachment
	Metadata    map[string]interface{}
}

// File represents a file in user input
type File struct {
	Path      string
	Name      string
	Extension string
	Size      int64
	MIMEType  string
	Content   []byte
	IsImage   bool
}

// Attachment represents an attachment (URL, base64, etc.)
type Attachment struct {
	Type     AttachmentType
	Content  string
	MIMEType string
	IsImage  bool
}

// AttachmentType categorizes attachments
type AttachmentType string

const (
	AttachmentFile   AttachmentType = "file"
	AttachmentURL    AttachmentType = "url"
	AttachmentBase64 AttachmentType = "base64"
)

// NewImageDetector creates a new image detector
func NewImageDetector(config *DetectionConfig) *ImageDetector {
	return &ImageDetector{
		methods:          config.Methods,
		contentInspector: NewContentInspector(),
		config:           config,
	}
}

// Detect checks input for images
func (d *ImageDetector) Detect(ctx context.Context, input *Input) (*DetectionResult, error) {
	result := &DetectionResult{
		HasImages: false,
		Images:    []*DetectedImage{},
	}

	// Check text for base64 images
	if input.Text != "" && d.hasMethod(DetectByBase64) {
		textImages, err := d.DetectBase64(input.Text)
		if err == nil && len(textImages) > 0 {
			result.Images = append(result.Images, textImages...)
			result.DetectionMethod = DetectByBase64
		}
	}

	// Check text for URLs
	if input.Text != "" && d.hasMethod(DetectByURL) {
		urlImages, err := d.DetectInText(input.Text)
		if err == nil && len(urlImages) > 0 {
			result.Images = append(result.Images, urlImages...)
			result.DetectionMethod = DetectByURL
		}
	}

	// Check files
	if input.Files != nil {
		for _, file := range input.Files {
			isImage, err := d.DetectInFile(file)
			if err == nil && isImage {
				detectedImage := &DetectedImage{
					Source:   SourceFile,
					Location: file.Path,
					Format:   file.Extension,
					Size:     file.Size,
					MIMEType: file.MIMEType,
					Valid:    true,
				}
				result.Images = append(result.Images, detectedImage)
			}
		}
	}

	// Check attachments
	if input.Attachments != nil {
		for _, attachment := range input.Attachments {
			if attachment.IsImage {
				source := SourceFile
				if attachment.Type == AttachmentURL {
					source = SourceURL
				} else if attachment.Type == AttachmentBase64 {
					source = SourceBase64
				}

				detectedImage := &DetectedImage{
					Source:   source,
					Location: attachment.Content,
					MIMEType: attachment.MIMEType,
					Valid:    true,
				}
				result.Images = append(result.Images, detectedImage)
			}
		}
	}

	result.ImageCount = len(result.Images)
	result.HasImages = result.ImageCount > 0
	result.Confidence = calculateConfidence(result)

	return result, nil
}

// DetectInText looks for image references in text
func (d *ImageDetector) DetectInText(text string) ([]*DetectedImage, error) {
	var images []*DetectedImage

	// Pattern for image URLs
	urlPattern := regexp.MustCompile(`https?://[^\s]+\.(jpg|jpeg|png|gif|webp|bmp)`)
	matches := urlPattern.FindAllString(text, -1)

	for _, match := range matches {
		images = append(images, &DetectedImage{
			Source:   SourceURL,
			Location: match,
			Format:   extractExtension(match),
			Valid:    true,
		})
	}

	return images, nil
}

// DetectInFile checks if a file is an image
func (d *ImageDetector) DetectInFile(file *File) (bool, error) {
	// Check by MIME type
	if d.hasMethod(DetectByMIME) && file.MIMEType != "" {
		if strings.HasPrefix(file.MIMEType, "image/") {
			file.IsImage = true
			return true, nil
		}
	}

	// Check by extension
	if d.hasMethod(DetectByExtension) && file.Extension != "" {
		ext := strings.ToLower(strings.TrimPrefix(file.Extension, "."))
		if d.isSupportedFormat(ext) {
			file.IsImage = true
			return true, nil
		}
	}

	// Check by content inspection
	if d.hasMethod(DetectByContent) && file.Content != nil && len(file.Content) > 0 {
		reader := bytes.NewReader(file.Content)
		inspectionResult, err := d.contentInspector.InspectContent(reader)
		if err == nil && inspectionResult.IsImage {
			file.IsImage = true
			file.MIMEType = inspectionResult.MIMEType
			return true, nil
		}
	}

	return false, nil
}

// DetectBase64 detects base64-encoded images
func (d *ImageDetector) DetectBase64(content string) ([]*DetectedImage, error) {
	var images []*DetectedImage

	// Pattern for base64 data URIs: data:image/png;base64,iVBORw0KGgo...
	base64Pattern := regexp.MustCompile(`data:image/([a-zA-Z]+);base64,([A-Za-z0-9+/=]+)`)
	matches := base64Pattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			format := match[1]
			data := match[2]

			// Validate base64
			if _, err := base64.StdEncoding.DecodeString(data); err == nil {
				images = append(images, &DetectedImage{
					Source:   SourceBase64,
					Location: "embedded",
					Format:   format,
					MIMEType: fmt.Sprintf("image/%s", format),
					Valid:    true,
				})
			}
		}
	}

	return images, nil
}

// ValidateImage validates that detected content is a valid image
func (d *ImageDetector) ValidateImage(reader io.Reader) (bool, string, error) {
	result, err := d.contentInspector.InspectContent(reader)
	if err != nil {
		return false, "", err
	}

	return result.IsImage, result.Format, nil
}

// hasMethod checks if a detection method is enabled
func (d *ImageDetector) hasMethod(method DetectionMethod) bool {
	for _, m := range d.methods {
		if m == method {
			return true
		}
	}
	return false
}

// isSupportedFormat checks if format is supported
func (d *ImageDetector) isSupportedFormat(format string) bool {
	format = strings.ToLower(format)
	for _, f := range d.config.SupportedFormats {
		if strings.ToLower(f) == format {
			return true
		}
	}
	return false
}

// ContentInspector performs deep content inspection
type ContentInspector struct {
	signatures map[string][]byte
}

// InspectionResult contains content inspection results
type InspectionResult struct {
	IsImage    bool
	Format     string
	MIMEType   string
	Confidence float64
	Dimensions *Dimensions
}

// Image format magic numbers (file signatures)
var ImageSignatures = map[string][]byte{
	"png":  {0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	"jpg":  {0xFF, 0xD8, 0xFF},
	"gif":  {0x47, 0x49, 0x46, 0x38},
	"webp": {0x52, 0x49, 0x46, 0x46}, // RIFF (WebP starts with RIFF)
	"bmp":  {0x42, 0x4D},
	"tiff": {0x49, 0x49, 0x2A, 0x00}, // Little-endian TIFF
}

// NewContentInspector creates a content inspector
func NewContentInspector() *ContentInspector {
	return &ContentInspector{
		signatures: ImageSignatures,
	}
}

// InspectContent checks file content for image signatures
func (c *ContentInspector) InspectContent(reader io.Reader) (*InspectionResult, error) {
	// Read first 16 bytes for magic number detection
	buf := make([]byte, 16)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	if n < 4 {
		return &InspectionResult{IsImage: false}, nil
	}

	// Check against known signatures
	for format, signature := range c.signatures {
		if n >= len(signature) && bytes.HasPrefix(buf, signature) {
			return &InspectionResult{
				IsImage:    true,
				Format:     format,
				MIMEType:   fmt.Sprintf("image/%s", format),
				Confidence: 1.0,
			}, nil
		}
	}

	return &InspectionResult{IsImage: false}, nil
}

// Helper functions

func extractExtension(path string) string {
	ext := filepath.Ext(path)
	return strings.ToLower(strings.TrimPrefix(ext, "."))
}

func calculateConfidence(result *DetectionResult) float64 {
	if result.ImageCount == 0 {
		return 0.0
	}

	validCount := 0
	for _, img := range result.Images {
		if img.Valid {
			validCount++
		}
	}

	if validCount == 0 {
		return 0.3
	}

	return float64(validCount) / float64(result.ImageCount)
}
