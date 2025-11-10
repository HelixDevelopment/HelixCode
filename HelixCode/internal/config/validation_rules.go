package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Built-in validation rules implementation

// CogneeMode represents Cognee operating mode
type CogneeMode string

const (
	CogneeModeLocal  CogneeMode = "local"
	CogneeModeRemote CogneeMode = "remote"
	CogneeModeHybrid CogneeMode = "hybrid"
	CogneeModeCloud  CogneeMode = "cloud"
)

// LoadBalancingStrategy represents load balancing strategy
type LoadBalancingStrategy string

const (
	LoadBalancingRoundRobin LoadBalancingStrategy = "round_robin"
	LoadBalancingLeastConn  LoadBalancingStrategy = "least_connections"
	LoadBalancingRandom     LoadBalancingStrategy = "random"
	LoadBalancingWeighted   LoadBalancingStrategy = "weighted"
)

// FallbackStrategy represents fallback strategy
type FallbackStrategy string

const (
	FallbackStrategyFailover FallbackStrategy = "failover"
	FallbackStrategyRetry    FallbackStrategy = "retry"
	FallbackStrategyCircuit  FallbackStrategy = "circuit_breaker"
)

// LengthRule validates string length
type LengthRule struct {
	MinLength int
	MaxLength int
}

func (r *LengthRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string for length validation")
	}

	length := len(str)
	if r.MinLength > 0 && length < r.MinLength {
		return fmt.Errorf("value length %d is less than minimum %d", length, r.MinLength)
	}

	if r.MaxLength > 0 && length > r.MaxLength {
		return fmt.Errorf("value length %d is greater than maximum %d", length, r.MaxLength)
	}

	return nil
}

func (r *LengthRule) GetName() string {
	return "length"
}

func (r *LengthRule) GetDescription() string {
	return fmt.Sprintf("Validates string length is between %d and %d", r.MinLength, r.MaxLength)
}

// RangeRule validates numeric range
type RangeRule struct {
	MinValue float64
	MaxValue float64
}

func (r *RangeRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	var num float64
	var err error

	switch v := value.(type) {
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float32:
		num = float64(v)
	case float64:
		num = v
	case string:
		num, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("cannot convert string to number for range validation: %w", err)
		}
	default:
		return fmt.Errorf("value must be numeric for range validation")
	}

	if num < r.MinValue {
		return fmt.Errorf("value %f is less than minimum %f", num, r.MinValue)
	}

	if num > r.MaxValue {
		return fmt.Errorf("value %f is greater than maximum %f", num, r.MaxValue)
	}

	return nil
}

func (r *RangeRule) GetName() string {
	return "range"
}

func (r *RangeRule) GetDescription() string {
	return fmt.Sprintf("Validates numeric value is between %f and %f", r.MinValue, r.MaxValue)
}

// RegexRule validates using regular expression
type RegexRule struct {
	Pattern  string
	Compiled *regexp.Regexp
}

func (r *RegexRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string for regex validation")
	}

	if r.Compiled == nil {
		compiled, err := regexp.Compile(r.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		r.Compiled = compiled
	}

	if !r.Compiled.MatchString(str) {
		return fmt.Errorf("value does not match pattern %s", r.Pattern)
	}

	return nil
}

func (r *RegexRule) GetName() string {
	return "regex"
}

func (r *RegexRule) GetDescription() string {
	return fmt.Sprintf("Validates value matches regex pattern: %s", r.Pattern)
}

// EmailRule validates email format
type EmailRule struct{}

func (r *EmailRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	email, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string for email validation")
	}

	// Basic email validation
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(emailRegex, email)
	if err != nil {
		return fmt.Errorf("email validation error: %w", err)
	}

	if !matched {
		return fmt.Errorf("value is not a valid email address")
	}

	return nil
}

func (r *EmailRule) GetName() string {
	return "email"
}

func (r *EmailRule) GetDescription() string {
	return "Validates email address format"
}

// URLRule validates URL format
type URLRule struct {
	Schemes []string // Allowed URL schemes, e.g., ["http", "https"]
}

func (r *URLRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	urlStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string for URL validation")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must include a scheme")
	}

	if len(r.Schemes) > 0 {
		schemeAllowed := false
		for _, scheme := range r.Schemes {
			if parsedURL.Scheme == scheme {
				schemeAllowed = true
				break
			}
		}
		if !schemeAllowed {
			return fmt.Errorf("URL scheme %s is not allowed", parsedURL.Scheme)
		}
	}

	return nil
}

func (r *URLRule) GetName() string {
	return "url"
}

func (r *URLRule) GetDescription() string {
	if len(r.Schemes) > 0 {
		return fmt.Sprintf("Validates URL format with allowed schemes: %v", r.Schemes)
	}
	return "Validates URL format"
}

// FileExistsRule validates file existence
type FileExistsRule struct {
	CreateIfMissing bool
	CheckReadable   bool
	CheckWritable   bool
}

func (r *FileExistsRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	filePath, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string for file existence validation")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if r.CreateIfMissing {
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			return nil
		}
		return fmt.Errorf("file or directory does not exist: %s", filePath)
	}

	if r.CheckReadable {
		if _, err := os.Open(filePath); err != nil {
			return fmt.Errorf("file or directory is not readable: %s", filePath)
		}
	}

	if r.CheckWritable {
		testFile := filepath.Join(filePath, ".helix_write_test")
		if file, err := os.Create(testFile); err != nil {
			return fmt.Errorf("file or directory is not writable: %s", filePath)
		} else {
			file.Close()
			os.Remove(testFile)
		}
	}

	return nil
}

func (r *FileExistsRule) GetName() string {
	return "file_exists"
}

func (r *FileExistsRule) GetDescription() string {
	return "Validates file or directory existence and permissions"
}

// EnumRule validates against allowed values
type EnumRule struct {
	AllowedValues []interface{}
	CaseSensitive bool
}

func (r *EnumRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	for _, allowed := range r.AllowedValues {
		if r.CaseSensitive {
			if reflect.DeepEqual(value, allowed) {
				return nil
			}
		} else {
			// Case-insensitive comparison for strings
			valueStr, ok := value.(string)
			allowedStr, ok := allowed.(string)
			if ok && strings.EqualFold(valueStr, allowedStr) {
				return nil
			}
		}
	}

	return fmt.Errorf("value %v is not in allowed values: %v", value, r.AllowedValues)
}

func (r *EnumRule) GetName() string {
	return "enum"
}

func (r *EnumRule) GetDescription() string {
	return fmt.Sprintf("Validates value is one of allowed values: %v", r.AllowedValues)
}

// TimeRule validates time format
type TimeRule struct {
	Format    string // time format string
	MinTime   *time.Time
	MaxTime   *time.Time
	AllowZero bool // allow zero time
}

func (r *TimeRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	var timeValue time.Time
	var err error

	switch v := value.(type) {
	case string:
		if r.Format != "" {
			timeValue, err = time.Parse(r.Format, v)
		} else {
			// Try common formats
			formats := []string{
				time.RFC3339,
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
				"2006-01-02",
			}
			for _, format := range formats {
				if timeValue, err = time.Parse(format, v); err == nil {
					break
				}
			}
		}
	case time.Time:
		timeValue = v
	default:
		return fmt.Errorf("value must be a time.Time or string for time validation")
	}

	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	if !r.AllowZero && timeValue.IsZero() {
		return fmt.Errorf("zero time is not allowed")
	}

	if r.MinTime != nil && timeValue.Before(*r.MinTime) {
		return fmt.Errorf("time %s is before minimum time %s", timeValue, *r.MinTime)
	}

	if r.MaxTime != nil && timeValue.After(*r.MaxTime) {
		return fmt.Errorf("time %s is after maximum time %s", timeValue, *r.MaxTime)
	}

	return nil
}

func (r *TimeRule) GetName() string {
	return "time"
}

func (r *TimeRule) GetDescription() string {
	return "Validates time format and range"
}

// IPAddressRule validates IP address format
type IPAddressRule struct {
	AllowIPv4 bool
	AllowIPv6 bool
}

func (r *IPAddressRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	ipStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string for IP address validation")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format: %s", ipStr)
	}

	if r.AllowIPv4 && ip.To4() != nil {
		return nil
	}

	if r.AllowIPv6 && ip.To4() == nil {
		return nil
	}

	if !r.AllowIPv4 && !r.AllowIPv6 {
		return nil // All IPs allowed
	}

	return fmt.Errorf("IP address type not allowed: %s", ipStr)
}

func (r *IPAddressRule) GetName() string {
	return "ip_address"
}

func (r *IPAddressRule) GetDescription() string {
	var types []string
	if r.AllowIPv4 {
		types = append(types, "IPv4")
	}
	if r.AllowIPv6 {
		types = append(types, "IPv6")
	}
	return fmt.Sprintf("Validates IP address format (allowed types: %s)", strings.Join(types, ", "))
}

// PortRule validates port number
type PortRule struct {
	MinPort int
	MaxPort int
}

func (r *PortRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	var port int
	var err error

	switch v := value.(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	case float64:
		port = int(v)
	case string:
		port, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("cannot convert string to port number: %w", err)
		}
	default:
		return fmt.Errorf("value must be numeric for port validation")
	}

	if port < 0 || port > 65535 {
		return fmt.Errorf("port number must be between 0 and 65535")
	}

	if r.MinPort > 0 && port < r.MinPort {
		return fmt.Errorf("port number %d is less than minimum %d", port, r.MinPort)
	}

	if r.MaxPort > 0 && port > r.MaxPort {
		return fmt.Errorf("port number %d is greater than maximum %d", port, r.MaxPort)
	}

	return nil
}

func (r *PortRule) GetName() string {
	return "port"
}

func (r *PortRule) GetDescription() string {
	return fmt.Sprintf("Validates port number is between %d and %d", r.MinPort, r.MaxPort)
}

// RequiredRule validates required fields
type RequiredRule struct{}

func (r *RequiredRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return fmt.Errorf("required field %s is missing", context.Property)
	}

	// Check for empty strings
	if str, ok := value.(string); ok && str == "" {
		return fmt.Errorf("required field %s is empty", context.Property)
	}

	// Check for empty collections
	if reflect.ValueOf(value).Kind() == reflect.Slice && reflect.ValueOf(value).Len() == 0 {
		return fmt.Errorf("required field %s is empty", context.Property)
	}

	return nil
}

func (r *RequiredRule) GetName() string {
	return "required"
}

func (r *RequiredRule) GetDescription() string {
	return "Validates that a required field is present and not empty"
}

// ConditionalRule validates based on conditions
type ConditionalRule struct {
	Condition string         // expression to evaluate
	Rule      ValidationRule // rule to apply if condition is true
	ElseRule  ValidationRule // rule to apply if condition is false
}

func (r *ConditionalRule) Validate(value interface{}, context *ValidationContext) error {
	conditionMet, err := r.evaluateCondition(context)
	if err != nil {
		return fmt.Errorf("failed to evaluate condition: %w", err)
	}

	if conditionMet {
		return r.Rule.Validate(value, context)
	} else if r.ElseRule != nil {
		return r.ElseRule.Validate(value, context)
	}

	return nil
}

func (r *ConditionalRule) GetName() string {
	return "conditional"
}

func (r *ConditionalRule) GetDescription() string {
	return fmt.Sprintf("Applies validation rule based on condition: %s", r.Condition)
}

func (r *ConditionalRule) evaluateCondition(context *ValidationContext) (bool, error) {
	// This would implement condition evaluation
	// For now, return true
	return true, nil
}

// CustomRule allows custom validation logic
type CustomRule struct {
	Name        string
	Description string
	Validator   func(interface{}, *ValidationContext) error
}

func (r *CustomRule) Validate(value interface{}, context *ValidationContext) error {
	return r.Validator(value, context)
}

func (r *CustomRule) GetName() string {
	return r.Name
}

func (r *CustomRule) GetDescription() string {
	return r.Description
}

// APIKeyRule validates API key format
type APIKeyRule struct {
	Prefix       string // expected prefix, e.g., "sk-"
	MinLength    int
	MaxLength    int
	AllowedChars string // regex for allowed characters
}

func (r *APIKeyRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	apiKey, ok := value.(string)
	if !ok {
		return fmt.Errorf("API key must be a string")
	}

	if r.Prefix != "" && !strings.HasPrefix(apiKey, r.Prefix) {
		return fmt.Errorf("API key must start with prefix %s", r.Prefix)
	}

	length := len(apiKey)
	if r.MinLength > 0 && length < r.MinLength {
		return fmt.Errorf("API key length %d is less than minimum %d", length, r.MinLength)
	}

	if r.MaxLength > 0 && length > r.MaxLength {
		return fmt.Errorf("API key length %d is greater than maximum %d", length, r.MaxLength)
	}

	if r.AllowedChars != "" {
		matched, err := regexp.MatchString("^"+r.AllowedChars+"*$", apiKey)
		if err != nil {
			return fmt.Errorf("invalid allowed characters pattern: %w", err)
		}
		if !matched {
			return fmt.Errorf("API key contains invalid characters")
		}
	}

	return nil
}

func (r *APIKeyRule) GetName() string {
	return "api_key"
}

func (r *APIKeyRule) GetDescription() string {
	return "Validates API key format and constraints"
}

// ProviderTypeRule validates provider type
type ProviderTypeRule struct {
	AllowedTypes []string
}

func (r *ProviderTypeRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	providerType, ok := value.(string)
	if !ok {
		return fmt.Errorf("provider type must be a string")
	}

	if len(r.AllowedTypes) > 0 {
		allowed := false
		for _, allowedType := range r.AllowedTypes {
			if providerType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("provider type %s is not allowed", providerType)
		}
	}

	return nil
}

func (r *ProviderTypeRule) GetName() string {
	return "provider_type"
}

func (r *ProviderTypeRule) GetDescription() string {
	return fmt.Sprintf("Validates provider type is one of allowed types: %v", r.AllowedTypes)
}

// CogneeModeRule validates Cognee mode
type CogneeModeRule struct {
	AllowedModes []CogneeMode
}

func (r *CogneeModeRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	modeStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("Cognee mode must be a string")
	}

	mode := CogneeMode(modeStr)
	if len(r.AllowedModes) > 0 {
		allowed := false
		for _, allowedMode := range r.AllowedModes {
			if mode == allowedMode {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("Cognee mode %s is not allowed", mode)
		}
	}

	return nil
}

func (r *CogneeModeRule) GetName() string {
	return "cognee_mode"
}

func (r *CogneeModeRule) GetDescription() string {
	return fmt.Sprintf("Validates Cognee mode is one of allowed modes: %v", r.AllowedModes)
}

// LoadBalancingStrategyRule validates load balancing strategy
type LoadBalancingStrategyRule struct {
	AllowedStrategies []LoadBalancingStrategy
}

func (r *LoadBalancingStrategyRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	strategyStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("load balancing strategy must be a string")
	}

	strategy := LoadBalancingStrategy(strategyStr)
	if len(r.AllowedStrategies) > 0 {
		allowed := false
		for _, allowedStrategy := range r.AllowedStrategies {
			if strategy == allowedStrategy {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("load balancing strategy %s is not allowed", strategy)
		}
	}

	return nil
}

func (r *LoadBalancingStrategyRule) GetName() string {
	return "load_balancing_strategy"
}

func (r *LoadBalancingStrategyRule) GetDescription() string {
	return fmt.Sprintf("Validates load balancing strategy is one of allowed strategies: %v", r.AllowedStrategies)
}

// FallbackStrategyRule validates fallback strategy
type FallbackStrategyRule struct {
	AllowedStrategies []FallbackStrategy
}

func (r *FallbackStrategyRule) Validate(value interface{}, context *ValidationContext) error {
	if value == nil {
		return nil
	}

	strategyStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("fallback strategy must be a string")
	}

	strategy := FallbackStrategy(strategyStr)
	if len(r.AllowedStrategies) > 0 {
		allowed := false
		for _, allowedStrategy := range r.AllowedStrategies {
			if strategy == allowedStrategy {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("fallback strategy %s is not allowed", strategy)
		}
	}

	return nil
}

func (r *FallbackStrategyRule) GetName() string {
	return "fallback_strategy"
}

func (r *FallbackStrategyRule) GetDescription() string {
	return fmt.Sprintf("Validates fallback strategy is one of allowed strategies: %v", r.AllowedStrategies)
}

// Utility functions for creating validation rules

// NewLengthRule creates a length validation rule
func NewLengthRule(min, max int) *LengthRule {
	return &LengthRule{
		MinLength: min,
		MaxLength: max,
	}
}

// NewRangeRule creates a range validation rule
func NewRangeRule(min, max float64) *RangeRule {
	return &RangeRule{
		MinValue: min,
		MaxValue: max,
	}
}

// NewRegexRule creates a regex validation rule
func NewRegexRule(pattern string) *RegexRule {
	return &RegexRule{
		Pattern: pattern,
	}
}

// NewEmailRule creates an email validation rule
func NewEmailRule() *EmailRule {
	return &EmailRule{}
}

// NewURLRule creates a URL validation rule
func NewURLRule(schemes ...string) *URLRule {
	return &URLRule{
		Schemes: schemes,
	}
}

// NewFileExistsRule creates a file existence validation rule
func NewFileExistsRule(createIfMissing, checkReadable, checkWritable bool) *FileExistsRule {
	return &FileExistsRule{
		CreateIfMissing: createIfMissing,
		CheckReadable:   checkReadable,
		CheckWritable:   checkWritable,
	}
}

// NewEnumRule creates an enum validation rule
func NewEnumRule(values ...interface{}) *EnumRule {
	return &EnumRule{
		AllowedValues: values,
		CaseSensitive: true,
	}
}

// NewTimeRule creates a time validation rule
func NewTimeRule(format string, minTime, maxTime *time.Time, allowZero bool) *TimeRule {
	return &TimeRule{
		Format:    format,
		MinTime:   minTime,
		MaxTime:   maxTime,
		AllowZero: allowZero,
	}
}

// NewIPAddressRule creates an IP address validation rule
func NewIPAddressRule(allowIPv4, allowIPv6 bool) *IPAddressRule {
	return &IPAddressRule{
		AllowIPv4: allowIPv4,
		AllowIPv6: allowIPv6,
	}
}

// NewPortRule creates a port validation rule
func NewPortRule(minPort, maxPort int) *PortRule {
	return &PortRule{
		MinPort: minPort,
		MaxPort: maxPort,
	}
}

// NewRequiredRule creates a required field validation rule
func NewRequiredRule() *RequiredRule {
	return &RequiredRule{}
}

// NewAPIKeyRule creates an API key validation rule
func NewAPIKeyRule(prefix string, minLength, maxLength int, allowedChars string) *APIKeyRule {
	return &APIKeyRule{
		Prefix:       prefix,
		MinLength:    minLength,
		MaxLength:    maxLength,
		AllowedChars: allowedChars,
	}
}

// NewProviderTypeRule creates a provider type validation rule
func NewProviderTypeRule(allowedTypes ...string) *ProviderTypeRule {
	return &ProviderTypeRule{
		AllowedTypes: allowedTypes,
	}
}

// NewCogneeModeRule creates a Cognee mode validation rule
func NewCogneeModeRule(allowedModes ...CogneeMode) *CogneeModeRule {
	return &CogneeModeRule{
		AllowedModes: allowedModes,
	}
}

// NewLoadBalancingStrategyRule creates a load balancing strategy validation rule
func NewLoadBalancingStrategyRule(allowedStrategies ...LoadBalancingStrategy) *LoadBalancingStrategyRule {
	return &LoadBalancingStrategyRule{
		AllowedStrategies: allowedStrategies,
	}
}

// NewFallbackStrategyRule creates a fallback strategy validation rule
func NewFallbackStrategyRule(allowedStrategies ...FallbackStrategy) *FallbackStrategyRule {
	return &FallbackStrategyRule{
		AllowedStrategies: allowedStrategies,
	}
}
