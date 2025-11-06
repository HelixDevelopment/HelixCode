package mapping

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Mapper generates codebase maps
type Mapper interface {
	// MapCodebase maps an entire codebase
	MapCodebase(ctx context.Context, root string, opts *MapOptions) (*CodebaseMap, error)

	// MapFile maps a single file
	MapFile(ctx context.Context, path string) (*FileMap, error)

	// MapFiles maps multiple files
	MapFiles(ctx context.Context, paths []string) ([]*FileMap, error)

	// UpdateMap updates a codebase map incrementally
	UpdateMap(ctx context.Context, cmap *CodebaseMap, changes []string) error

	// GetDefinitions extracts definitions from code
	GetDefinitions(ctx context.Context, path string) ([]*Definition, error)

	// GetReferences finds references to a definition
	GetReferences(ctx context.Context, def *Definition, scope *CodebaseMap) ([]*Reference, error)
}

// DefaultMapper implements Mapper
type DefaultMapper struct {
	parser         TreeSitterParser
	registry       LanguageRegistry
	cache          CacheManager
	tokenCounter   *TokenCounter
	importAnalyzer *ImportAnalyzer
}

// NewDefaultMapper creates a new mapper
func NewDefaultMapper(
	parser TreeSitterParser,
	registry LanguageRegistry,
	cache CacheManager,
) *DefaultMapper {
	return &DefaultMapper{
		parser:         parser,
		registry:       registry,
		cache:          cache,
		tokenCounter:   NewTokenCounter(),
		importAnalyzer: NewImportAnalyzer(),
	}
}

// NewMapper creates a new mapper with default components
func NewMapper(workspaceRoot string) *DefaultMapper {
	registry := NewDefaultLanguageRegistry()
	cache := NewDiskCacheManager(workspaceRoot)
	parser := NewDefaultTreeSitterParser(registry)

	return NewDefaultMapper(parser, registry, cache)
}

// MapCodebase maps an entire codebase
func (m *DefaultMapper) MapCodebase(ctx context.Context, root string, opts *MapOptions) (*CodebaseMap, error) {
	if opts == nil {
		opts = DefaultMapOptions()
	}

	// Try to load from cache
	if opts.UseCache {
		if cached, err := m.cache.Load(root); err == nil {
			// Check if cache is still valid
			if m.isCacheValid(cached, root) {
				return cached, nil
			}
		}
	}

	// Create new map
	cmap := NewCodebaseMap(root)

	// Find all source files
	files, err := m.findSourceFiles(root, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find source files: %w", err)
	}

	// Map files concurrently
	fileMaps := make(chan *FileMap, len(files))
	errors := make(chan error, len(files))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, opts.Concurrency)

	for _, file := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check context
			select {
			case <-ctx.Done():
				errors <- ctx.Err()
				return
			default:
			}

			fileMap, err := m.MapFile(ctx, path)
			if err != nil {
				// Log error but don't fail entire operation
				errors <- fmt.Errorf("failed to map %s: %w", path, err)
				return
			}

			fileMaps <- fileMap
		}(file)
	}

	go func() {
		wg.Wait()
		close(fileMaps)
		close(errors)
	}()

	// Collect results
	for fileMap := range fileMaps {
		cmap.AddFile(fileMap)
	}

	// Collect errors (non-blocking)
	var mapErrors []error
	for err := range errors {
		if err != nil {
			mapErrors = append(mapErrors, err)
		}
	}

	// Build dependencies
	for path, fileMap := range cmap.Files {
		deps := m.importAnalyzer.ResolveDependencies(fileMap, cmap)
		cmap.Dependencies[path] = deps
	}

	cmap.UpdatedAt = cmap.CreatedAt

	// Save to cache
	if opts.UseCache {
		if err := m.cache.Save(cmap); err != nil {
			// Log warning but don't fail
			// In production, you'd use proper logging
		}
	}

	// Return error if any mappings failed
	if len(mapErrors) > 0 {
		return cmap, fmt.Errorf("mapped with %d errors (first: %w)", len(mapErrors), mapErrors[0])
	}

	return cmap, nil
}

// MapFile maps a single file
func (m *DefaultMapper) MapFile(ctx context.Context, path string) (*FileMap, error) {
	// Try to load from cache
	if cached, err := m.cache.LoadFile(path); err == nil {
		return cached, nil
	}

	// Read file
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect language
	language := DetectLanguage(path)
	if language == "" {
		return nil, fmt.Errorf("unsupported language for file: %s", path)
	}

	// Parse file
	tree, err := m.parser.Parse(ctx, source, language)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Get language parser
	langParser, err := m.registry.Get(language)
	if err != nil {
		// If no language parser is registered, create a basic file map
		return m.createBasicFileMap(path, language, source), nil
	}

	// Extract definitions
	definitions, err := langParser.ExtractDefinitions(tree)
	if err != nil {
		definitions = []*Definition{}
	}

	// Update definition file paths and qualified names
	for _, def := range definitions {
		def.FilePath = path
		if def.QualifiedName == "" {
			def.QualifiedName = m.generateQualifiedName(def, path, language)
		}
	}

	// Extract imports
	imports, err := langParser.ExtractImports(tree)
	if err != nil {
		imports = []*Import{}
	}

	// Extract comments
	comments := tree.ExtractComments()

	// Calculate complexity
	complexity := langParser.CalculateComplexity(tree)

	// Count tokens
	tokens := m.tokenCounter.Count(source, language)

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Create file map
	fileMap := &FileMap{
		Path:        path,
		Language:    language,
		Size:        info.Size(),
		Lines:       CountLines(source),
		Tokens:      tokens,
		Definitions: definitions,
		Imports:     imports,
		Comments:    comments,
		Complexity:  complexity,
		Checksum:    CalculateChecksum(source),
		ParsedAt:    info.ModTime(),
	}

	// Save to cache
	_ = m.cache.SaveFile(fileMap)

	return fileMap, nil
}

// MapFiles maps multiple files
func (m *DefaultMapper) MapFiles(ctx context.Context, paths []string) ([]*FileMap, error) {
	fileMaps := make([]*FileMap, 0, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := make(chan error, len(paths))

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			fileMap, err := m.MapFile(ctx, p)
			if err != nil {
				errors <- err
				return
			}

			mu.Lock()
			fileMaps = append(fileMaps, fileMap)
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fileMaps, fmt.Errorf("failed to map %d files", len(errs))
	}

	return fileMaps, nil
}

// UpdateMap updates a codebase map incrementally
func (m *DefaultMapper) UpdateMap(ctx context.Context, cmap *CodebaseMap, changes []string) error {
	for _, path := range changes {
		// Check if file still exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// File was deleted
			cmap.RemoveFile(path)
			continue
		}

		// Re-map the file
		fileMap, err := m.MapFile(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to update file %s: %w", path, err)
		}

		// Remove old version and add new
		cmap.RemoveFile(path)
		cmap.AddFile(fileMap)

		// Update dependencies
		deps := m.importAnalyzer.ResolveDependencies(fileMap, cmap)
		cmap.Dependencies[path] = deps
	}

	cmap.UpdatedAt = cmap.CreatedAt

	// Save to cache
	if err := m.cache.Save(cmap); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// GetDefinitions extracts definitions from code
func (m *DefaultMapper) GetDefinitions(ctx context.Context, path string) ([]*Definition, error) {
	fileMap, err := m.MapFile(ctx, path)
	if err != nil {
		return nil, err
	}
	return fileMap.Definitions, nil
}

// GetReferences finds references to a definition
func (m *DefaultMapper) GetReferences(ctx context.Context, def *Definition, scope *CodebaseMap) ([]*Reference, error) {
	// This is a simplified implementation
	// Production code would use tree-sitter queries to find references

	var references []*Reference

	// Search through all files in the scope
	for path, fileMap := range scope.Files {
		if path == def.FilePath {
			continue // Skip the definition file
		}

		// Read file
		source, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Simple text search for the definition name
		lines := strings.Split(string(source), "\n")
		for i, line := range lines {
			if strings.Contains(line, def.Name) {
				references = append(references, &Reference{
					DefinitionID: def.QualifiedName,
					FilePath:     path,
					Line:         i + 1,
					Column:       strings.Index(line, def.Name),
					Context:      strings.TrimSpace(line),
				})
			}
		}

		_ = fileMap // Silence unused variable warning
	}

	return references, nil
}

// findSourceFiles finds all source files in a directory
func (m *DefaultMapper) findSourceFiles(root string, opts *MapOptions) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory should be excluded
			if m.shouldExcludeDir(path, root, opts.ExcludeDirs) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files unless explicitly included
		if !opts.IncludeHidden && strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check if file should be included
		if m.shouldIncludeFile(path, info, opts) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// shouldExcludeDir checks if a directory should be excluded
func (m *DefaultMapper) shouldExcludeDir(path, root string, excludeDirs []string) bool {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	parts := strings.Split(relPath, string(filepath.Separator))

	for _, part := range parts {
		for _, exclude := range excludeDirs {
			if part == exclude {
				return true
			}
		}
	}

	return false
}

// shouldIncludeFile checks if a file should be included
func (m *DefaultMapper) shouldIncludeFile(path string, info os.FileInfo, opts *MapOptions) bool {
	// Check file size
	if info.Size() > opts.MaxFileSize {
		return false
	}

	// Check if language is supported
	lang := DetectLanguage(path)
	if lang == "" {
		return false
	}

	// Filter by languages if specified
	if len(opts.Languages) > 0 {
		found := false
		for _, l := range opts.Languages {
			if strings.EqualFold(l, lang) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// isCacheValid checks if a cached map is still valid
func (m *DefaultMapper) isCacheValid(cached *CodebaseMap, root string) bool {
	// Simple check: verify a few random files haven't changed
	// Production code would be more sophisticated

	checked := 0
	for path, fileMap := range cached.Files {
		if checked >= 10 {
			break
		}
		checked++

		info, err := os.Stat(path)
		if err != nil {
			return false
		}

		// Check size
		if info.Size() != fileMap.Size {
			return false
		}

		// Check modification time
		if info.ModTime().After(fileMap.ParsedAt) {
			return false
		}
	}

	return true
}

// generateQualifiedName generates a qualified name for a definition
func (m *DefaultMapper) generateQualifiedName(def *Definition, path, language string) string {
	// Generate a qualified name based on file path and definition name
	// Format: package.class.method or file::function

	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	if def.Parent != "" {
		return fmt.Sprintf("%s.%s.%s", name, def.Parent, def.Name)
	}

	return fmt.Sprintf("%s.%s", name, def.Name)
}

// createBasicFileMap creates a basic file map when parsing fails
func (m *DefaultMapper) createBasicFileMap(path, language string, source []byte) *FileMap {
	info, _ := os.Stat(path)

	return &FileMap{
		Path:        path,
		Language:    language,
		Size:        int64(len(source)),
		Lines:       CountLines(source),
		Tokens:      m.tokenCounter.Count(source, language),
		Definitions: []*Definition{},
		Imports:     []*Import{},
		Comments:    []*Comment{},
		Complexity:  1,
		Checksum:    CalculateChecksum(source),
		ParsedAt:    info.ModTime(),
	}
}
