package config

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Built-in transformers implementation

// EnvVarTransformer substitutes environment variables
type EnvVarTransformer struct {
	Prefix     string // Environment variable prefix, e.g., "HELIX_"
	Default    string // Default value if environment variable not found
	Required    bool   // Whether the environment variable is required
}

func (t *EnvVarTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Find environment variable placeholders
	placeholderRegex := regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)
	matches := placeholderRegex.FindAllStringSubmatch(str, -1)

	result := str
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		envVar := match[1]
		if t.Prefix != "" {
			envVar = t.Prefix + envVar
		}

		envValue := os.Getenv(envVar)
		if envValue == "" {
			if t.Required {
				return nil, fmt.Errorf("required environment variable %s is not set", envVar)
			}
			if t.Default != "" {
				envValue = t.Default
			}
		}

		result = strings.ReplaceAll(result, match[0], envValue)
	}

	return result, nil
}

func (t *EnvVarTransformer) GetName() string {
	return "env_var"
}

func (t *EnvVarTransformer) GetDescription() string {
	return "Substitutes environment variables in configuration values"
}

// PathTransformer resolves and normalizes file paths
type PathTransformer struct {
	BasePath      string // Base path for relative paths
	ExpandUser    bool   // Expand user home directory (~)
	ExpandEnv     bool   // Expand environment variables
	Absolutize    bool   // Convert to absolute paths
	Normalize     bool   // Normalize path separators
	CreateMissing bool   // Create missing directories
}

func (t *PathTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	pathStr, ok := value.(string)
	if !ok {
		return value, nil
	}

	result := pathStr

	// Expand environment variables
	if t.ExpandEnv {
		result = os.ExpandEnv(result)
	}

	// Expand user home directory
	if t.ExpandUser && strings.HasPrefix(result, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		result = strings.Replace(result, "~", home, 1)
	}

	// Make path absolute
	if t.Absolutize && !filepath.IsAbs(result) {
		basePath := t.BasePath
		if basePath == "" {
			// Use current working directory
			wd, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get working directory: %w", err)
			}
			basePath = wd
		}
		result = filepath.Join(basePath, result)
	}

	// Normalize path
	if t.Normalize {
		result = filepath.Clean(result)
	}

	// Create missing directories
	if t.CreateMissing {
		if err := os.MkdirAll(result, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", result, err)
		}
	}

	return result, nil
}

func (t *PathTransformer) GetName() string {
	return "path"
}

func (t *PathTransformer) GetDescription() string {
	return "Resolves and normalizes file paths"
}

// URLTransformer validates and normalizes URLs
type URLTransformer struct {
	DefaultScheme string // Default scheme if not provided
	ForceHTTPS     bool   // Force HTTPS scheme
	NormalizePath  bool   // Normalize URL path
	AddTrailingSlash bool // Add trailing slash to path
}

func (t *URLTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	urlStr, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Add default scheme if missing
	if !strings.Contains(urlStr, "://") {
		if t.DefaultScheme != "" {
			urlStr = t.DefaultScheme + "://" + urlStr
		} else {
			return nil, fmt.Errorf("URL missing scheme and no default scheme provided")
		}
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	// Force HTTPS
	if t.ForceHTTPS && parsedURL.Scheme != "https" {
		parsedURL.Scheme = "https"
	}

	// Normalize path
	if t.NormalizePath {
		parsedURL.Path = filepath.ToSlash(parsedURL.Path)
		if !strings.HasPrefix(parsedURL.Path, "/") {
			parsedURL.Path = "/" + parsedURL.Path
		}
	}

	// Add trailing slash
	if t.AddTrailingSlash && !strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path = parsedURL.Path + "/"
	}

	return parsedURL.String(), nil
}

func (t *URLTransformer) GetName() string {
	return "url"
}

func (t *URLTransformer) GetDescription() string {
	return "Validates and normalizes URLs"
}

// DurationTransformer converts string duration to time.Duration
type DurationTransformer struct {
	DefaultUnit string // Default time unit
	AllowZero   bool   // Allow zero duration
}

func (t *DurationTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	var duration time.Duration
	var err error

	switch v := value.(type) {
	case string:
		// Parse duration string
		duration, err = time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid duration format: %w", err)
		}
	case int, int64, float32, float64:
		// Convert numeric value with default unit
		num := fmt.Sprintf("%v", v)
		if t.DefaultUnit != "" {
			duration, err = time.ParseDuration(num + t.DefaultUnit)
		} else {
			duration, err = time.ParseDuration(num)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid numeric duration: %w", err)
		}
	case time.Duration:
		duration = v
	default:
		return value, nil // No transformation for other types
	}

	if !t.AllowZero && duration == 0 {
		return nil, fmt.Errorf("zero duration is not allowed")
	}

	return duration, nil
}

func (t *DurationTransformer) GetName() string {
	return "duration"
}

func (t *DurationTransformer) GetDescription() string {
	return "Converts string or numeric values to time.Duration"
}

// BooleanTransformer converts various boolean representations
type BooleanTransformer struct {
	TrueValues  []string // Values that represent true
	FalseValues []string // Values that represent false
	CaseSensitive bool  // Whether string matching is case sensitive
}

func (t *BooleanTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		str := v
		if !t.CaseSensitive {
			str = strings.ToLower(str)
		}

		// Check true values
		for _, trueVal := range t.TrueValues {
			if !t.CaseSensitive {
				trueVal = strings.ToLower(trueVal)
			}
			if str == trueVal {
				return true, nil
			}
		}

		// Check false values
		for _, falseVal := range t.FalseValues {
			if !t.CaseSensitive {
				falseVal = strings.ToLower(falseVal)
			}
			if str == falseVal {
				return false, nil
			}
		}

		return nil, fmt.Errorf("cannot convert string '%s' to boolean", v)
	case int, int64, float32, float64:
		// Non-zero numbers are true
		num := fmt.Sprintf("%v", v)
		if num == "0" {
			return false, nil
		} else {
			return true, nil
		}
	default:
		return nil, fmt.Errorf("cannot convert %T to boolean", value)
	}
}

func (t *BooleanTransformer) GetName() string {
	return "boolean"
}

func (t *BooleanTransformer) GetDescription() string {
	return "Converts various representations to boolean values"
}

// TemplateTransformer applies template substitution
type TemplateTransformer struct {
	Variables map[string]interface{} // Template variables
	Delimiters []string              // Template delimiters [left, right]
}

func (t *TemplateTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	templateStr, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Set default delimiters
	leftDelim := "{{"
	rightDelim := "}}"
	if len(t.Delimiters) == 2 {
		leftDelim = t.Delimiters[0]
		rightDelim = t.Delimiters[1]
	}

	// Merge variables
	variables := make(map[string]interface{})
	
	// Add context variables
	if context.Environment != nil {
		for k, v := range context.Environment {
			variables[k] = v
		}
	}
	
	// Add transformer variables
	for k, v := range t.Variables {
		variables[k] = v
	}
	
	// Add special variables
	variables["timestamp"] = time.Now().Format(time.RFC3339)
	variables["property"] = context.Property
	variables["path"] = context.FullPath

	// Simple template substitution
	result := templateStr
	for key, val := range variables {
		placeholder := leftDelim + key + rightDelim
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
	}

	return result, nil
}

func (t *TemplateTransformer) GetName() string {
	return "template"
}

func (t *TemplateTransformer) GetDescription() string {
	return "Applies template substitution with variables"
}

// HashTransformer generates hash values
type HashTransformer struct {
	Algorithm  string // Hash algorithm: md5, sha256
	Format     string // Output format: hex, base64
	Salt       string // Optional salt for hashing
}

func (t *HashTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Add salt if provided
	if t.Salt != "" {
		str = str + t.Salt
	}

	var hash []byte
	switch strings.ToLower(t.Algorithm) {
	case "md5":
		h := md5.Sum([]byte(str))
		hash = h[:]
	case "sha256":
		h := sha256.Sum256([]byte(str))
		hash = h[:]
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", t.Algorithm)
	}

	// Format output
	switch strings.ToLower(t.Format) {
	case "hex":
		return fmt.Sprintf("%x", hash), nil
	case "base64":
		return base64.StdEncoding.EncodeToString(hash), nil
	default:
		return nil, fmt.Errorf("unsupported hash format: %s", t.Format)
	}
}

func (t *HashTransformer) GetName() string {
	return "hash"
}

func (t *HashTransformer) GetDescription() string {
	return "Generates hash values for strings"
}

// Base64Transformer encodes/decodes base64 values
type Base64Transformer struct {
	Encode bool   // Whether to encode (true) or decode (false)
	URLSafe bool   // Whether to use URL-safe encoding
	Padding bool   // Whether to include padding
}

func (t *Base64Transformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	if t.Encode {
		// Encode to base64
		data := []byte(str)
		if t.URLSafe {
			str = base64.URLEncoding.EncodeToString(data)
		} else {
			str = base64.StdEncoding.EncodeToString(data)
		}
		
		if !t.Padding {
			str = strings.TrimRight(str, "=")
		}
	} else {
		// Decode from base64
		var data []byte
		var err error
		
		if t.URLSafe {
			// Add padding if missing
			if len(str)%4 != 0 {
				str += strings.Repeat("=", 4-len(str)%4)
			}
			data, err = base64.URLEncoding.DecodeString(str)
		} else {
			data, err = base64.StdEncoding.DecodeString(str)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
		
		str = string(data)
	}

	return str, nil
}

func (t *Base64Transformer) GetName() string {
	return "base64"
}

func (t *Base64Transformer) GetDescription() string {
	return "Encodes or decodes base64 values"
}

// RegexTransformer applies regex substitution
type RegexTransformer struct {
	Pattern    string // Regular expression pattern
	Replacement string // Replacement string
	Flags      string // Regex flags (i, m, s, etc.)
}

func (t *RegexTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	// Compile regex with flags
	flags := 0
	if strings.Contains(t.Flags, "i") {
		flags |= regexp.IGNORECASE
	}
	if strings.Contains(t.Flags, "m") {
		flags |= regexp.MULTILINE
	}
	if strings.Contains(t.Flags, "s") {
		flags |= regexp.DOTALL
	}

	re, err := regexp.Compile(t.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Apply substitution
	result := re.ReplaceAllString(str, t.Replacement)
	return result, nil
}

func (t *RegexTransformer) GetName() string {
	return "regex"
}

func (t *RegexTransformer) GetDescription() string {
	return "Applies regular expression substitution"
}

// SplitTransformer splits strings into arrays
type SplitTransformer struct {
	Separator string // Separator for splitting
	Trim      bool   // Trim whitespace from parts
	Filter     bool   // Filter out empty parts
	Limit      int    // Maximum number of splits
}

func (t *SplitTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	str, ok := value.(string)
	if !ok {
		return value, nil
	}

	var parts []string
	if t.Separator == "" {
		parts = strings.Split(str, "")
	} else {
		if t.Limit > 0 {
			parts = strings.SplitN(str, t.Separator, t.Limit)
		} else {
			parts = strings.Split(str, t.Separator)
		}
	}

	// Process parts
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if t.Trim {
			part = strings.TrimSpace(part)
		}
		
		if t.Filter && part == "" {
			continue
		}
		
		result = append(result, part)
	}

	return result, nil
}

func (t *SplitTransformer) GetName() string {
	return "split"
}

func (t *SplitTransformer) GetDescription() string {
	return "Splits strings into arrays"
}

// JoinTransformer joins arrays into strings
type JoinTransformer struct {
	Separator string // Separator for joining
}

func (t *JoinTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	arr, ok := value.([]string)
	if !ok {
		// Try to convert to string array
		if reflectValue := reflect.ValueOf(value); reflectValue.Kind() == reflect.Slice {
			arr = make([]string, reflectValue.Len())
			for i := 0; i < reflectValue.Len(); i++ {
				arr[i] = fmt.Sprintf("%v", reflectValue.Index(i).Interface())
			}
		} else {
			return value, nil
		}
	}

	return strings.Join(arr, t.Separator), nil
}

func (t *JoinTransformer) GetName() string {
	return "join"
}

func (t *JoinTransformer) GetDescription() string {
	return "Joins arrays into strings"
}

// NumericTransformer performs numeric operations
type NumericTransformer struct {
	Operation string // Operation: add, subtract, multiply, divide, pow
	Value     float64 // Value to use in operation
	Round     int     // Decimal places to round to
}

func (t *NumericTransformer) Transform(value interface{}, context *TransformContext) (interface{}, error) {
	if value == nil {
		return value, nil
	}

	var num float64
	var err error

	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		switch val := v.(type) {
		case int:
			num = float64(val)
		case int8:
			num = float64(val)
		case int16:
			num = float64(val)
		case int32:
			num = float64(val)
		case int64:
			num = float64(val)
		}
	case uint, uint8, uint16, uint32, uint64:
		switch val := v.(type) {
		case uint:
			num = float64(val)
		case uint8:
			num = float64(val)
		case uint16:
			num = float64(val)
		case uint32:
			num = float64(val)
		case uint64:
			num = float64(val)
		}
	case float32, float64:
		switch val := v.(type) {
		case float32:
			num = float64(val)
		case float64:
			num = val
		}
	case string:
		num, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert string to number: %w", err)
		}
	default:
		return value, nil // No transformation for other types
	}

	// Apply operation
	switch strings.ToLower(t.Operation) {
	case "add":
		num += t.Value
	case "subtract":
		num -= t.Value
	case "multiply":
		num *= t.Value
	case "divide":
		if t.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		num /= t.Value
	case "pow":
		num = math.Pow(num, t.Value)
	default:
		return nil, fmt.Errorf("unsupported numeric operation: %s", t.Operation)
	}

	// Round if specified
	if t.Round >= 0 {
		multiplier := math.Pow10(t.Round)
		num = math.Round(num*multiplier) / multiplier
	}

	return num, nil
}

func (t *NumericTransformer) GetName() string {
	return "numeric"
}

func (t *NumericTransformer) GetDescription() string {
	return "Performs numeric operations on values"
}

// Utility functions for creating transformers

// NewEnvVarTransformer creates an environment variable transformer
func NewEnvVarTransformer(prefix, defaultValue string, required bool) *EnvVarTransformer {
	return &EnvVarTransformer{
		Prefix:  prefix,
		Default: defaultValue,
		Required: required,
	}
}

// NewPathTransformer creates a path transformer
func NewPathTransformer(basePath string, expandUser, expandEnv, absolutize, normalize, createMissing bool) *PathTransformer {
	return &PathTransformer{
		BasePath:       basePath,
		ExpandUser:     expandUser,
		ExpandEnv:      expandEnv,
		Absolutize:    absolutize,
		Normalize:      normalize,
		CreateMissing:  createMissing,
	}
}

// NewURLTransformer creates a URL transformer
func NewURLTransformer(defaultScheme string, forceHTTPS, normalizePath, addTrailingSlash bool) *URLTransformer {
	return &URLTransformer{
		DefaultScheme:   defaultScheme,
		ForceHTTPS:      forceHTTPS,
		NormalizePath:   normalizePath,
		AddTrailingSlash: addTrailingSlash,
	}
}

// NewDurationTransformer creates a duration transformer
func NewDurationTransformer(defaultUnit string, allowZero bool) *DurationTransformer {
	return &DurationTransformer{
		DefaultUnit: defaultUnit,
		AllowZero:   allowZero,
	}
}

// NewBooleanTransformer creates a boolean transformer
func NewBooleanTransformer(trueValues, falseValues []string, caseSensitive bool) *BooleanTransformer {
	return &BooleanTransformer{
		TrueValues:   trueValues,
		FalseValues:  falseValues,
		CaseSensitive: caseSensitive,
	}
}

// NewTemplateTransformer creates a template transformer
func NewTemplateTransformer(variables map[string]interface{}, delimiters []string) *TemplateTransformer {
	return &TemplateTransformer{
		Variables: variables,
		Delimiters: delimiters,
	}
}

// NewHashTransformer creates a hash transformer
func NewHashTransformer(algorithm, format, salt string) *HashTransformer {
	return &HashTransformer{
		Algorithm: algorithm,
		Format:    format,
		Salt:      salt,
	}
}

// NewBase64Transformer creates a base64 transformer
func NewBase64Transformer(encode, urlSafe, padding bool) *Base64Transformer {
	return &Base64Transformer{
		Encode:  encode,
		URLSafe:  urlSafe,
		Padding: padding,
	}
}

// NewRegexTransformer creates a regex transformer
func NewRegexTransformer(pattern, replacement, flags string) *RegexTransformer {
	return &RegexTransformer{
		Pattern:    pattern,
		Replacement: replacement,
		Flags:      flags,
	}
}

// NewSplitTransformer creates a split transformer
func NewSplitTransformer(separator string, trim, filter bool, limit int) *SplitTransformer {
	return &SplitTransformer{
		Separator: separator,
		Trim:      trim,
		Filter:    filter,
		Limit:     limit,
	}
}

// NewJoinTransformer creates a join transformer
func NewJoinTransformer(separator string) *JoinTransformer {
	return &JoinTransformer{
		Separator: separator,
	}
}

// NewNumericTransformer creates a numeric transformer
func NewNumericTransformer(operation string, value float64, round int) *NumericTransformer {
	return &NumericTransformer{
		Operation: operation,
		Value:     value,
		Round:     round,
	}
}

// Import required for math operations
import "math"