package repomap

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileScore represents the relevance score of a file
type FileScore struct {
	FilePath string
	Score    float64
	Reasons  []string // Reasons for the score (for debugging)
}

// FileRanker ranks files by relevance to a query
type FileRanker struct {
	weights RankingWeights
}

// RankingWeights defines the weights for different ranking factors
type RankingWeights struct {
	RecentlyChanged float64
	SymbolMatch     float64
	ImportFrequency float64
	DependencyDepth float64
	FileSize        float64
	SymbolDensity   float64
}

// DefaultWeights returns sensible default weights
func DefaultWeights() RankingWeights {
	return RankingWeights{
		RecentlyChanged: 0.3,
		SymbolMatch:     0.4,
		ImportFrequency: 0.1,
		DependencyDepth: 0.1,
		FileSize:        0.05,
		SymbolDensity:   0.05,
	}
}

// NewFileRanker creates a new file ranker
func NewFileRanker() *FileRanker {
	return &FileRanker{
		weights: DefaultWeights(),
	}
}

// NewFileRankerWithWeights creates a file ranker with custom weights
func NewFileRankerWithWeights(weights RankingWeights) *FileRanker {
	return &FileRanker{
		weights: weights,
	}
}

// RankFiles ranks files based on relevance to a query
func (fr *FileRanker) RankFiles(symbols []Symbol, query string, changedFiles []string) []FileScore {
	// Group symbols by file
	symbolsByFile := make(map[string][]Symbol)
	for _, symbol := range symbols {
		symbolsByFile[symbol.FilePath] = append(symbolsByFile[symbol.FilePath], symbol)
	}

	// Calculate scores
	scores := make([]FileScore, 0)
	queryTerms := fr.tokenizeQuery(query)

	for filePath, fileSymbols := range symbolsByFile {
		score := 0.0
		reasons := make([]string, 0)

		// 1. Recently changed files get higher priority
		recentScore := fr.scoreRecentChange(filePath, changedFiles)
		if recentScore > 0 {
			score += recentScore * fr.weights.RecentlyChanged
			reasons = append(reasons, "recently changed")
		}

		// 2. Symbol relevance to query
		symbolScore := fr.scoreSymbolRelevance(fileSymbols, queryTerms)
		if symbolScore > 0 {
			score += symbolScore * fr.weights.SymbolMatch
			reasons = append(reasons, "symbol match")
		}

		// 3. Import/dependency relationships
		importScore := fr.scoreImportFrequency(filePath, fileSymbols, symbolsByFile)
		if importScore > 0 {
			score += importScore * fr.weights.ImportFrequency
			reasons = append(reasons, "high import frequency")
		}

		// 4. Dependency depth (files that many others depend on)
		depthScore := fr.scoreDependencyDepth(filePath, symbolsByFile)
		if depthScore > 0 {
			score += depthScore * fr.weights.DependencyDepth
			reasons = append(reasons, "central dependency")
		}

		// 5. File size (prefer smaller, focused files)
		sizeScore := fr.scoreFileSize(filePath)
		score += sizeScore * fr.weights.FileSize

		// 6. Symbol density (more symbols = more important)
		densityScore := fr.scoreSymbolDensity(fileSymbols, filePath)
		if densityScore > 0 {
			score += densityScore * fr.weights.SymbolDensity
			reasons = append(reasons, "high symbol density")
		}

		scores = append(scores, FileScore{
			FilePath: filePath,
			Score:    score,
			Reasons:  reasons,
		})
	}

	return scores
}

// scoreRecentChange scores based on whether file was recently changed
func (fr *FileRanker) scoreRecentChange(filePath string, changedFiles []string) float64 {
	for _, changed := range changedFiles {
		if changed == filePath {
			return 1.0
		}
	}

	// Also consider file modification time
	info, err := os.Stat(filePath)
	if err != nil {
		return 0.0
	}

	// Files modified in the last hour get a boost
	age := time.Since(info.ModTime())
	if age < time.Hour {
		return 0.8
	} else if age < 24*time.Hour {
		return 0.5
	} else if age < 7*24*time.Hour {
		return 0.2
	}

	return 0.0
}

// scoreSymbolRelevance scores based on how well symbols match the query
func (fr *FileRanker) scoreSymbolRelevance(symbols []Symbol, queryTerms []string) float64 {
	if len(queryTerms) == 0 {
		return 0.0
	}

	matchCount := 0
	totalWeight := 0.0

	for _, symbol := range symbols {
		symbolTerms := fr.tokenizeSymbolName(symbol.Name)

		for _, queryTerm := range queryTerms {
			for _, symbolTerm := range symbolTerms {
				if strings.Contains(strings.ToLower(symbolTerm), strings.ToLower(queryTerm)) {
					matchCount++

					// Weight by symbol type
					weight := fr.getSymbolTypeWeight(symbol.Type)
					totalWeight += weight
				}
			}
		}

		// Also check docstrings
		if symbol.Docstring != "" {
			docLower := strings.ToLower(symbol.Docstring)
			for _, queryTerm := range queryTerms {
				if strings.Contains(docLower, strings.ToLower(queryTerm)) {
					totalWeight += 0.3 // Docstring matches are less important
				}
			}
		}
	}

	if matchCount == 0 {
		return 0.0
	}

	// Normalize by number of query terms and symbols
	score := totalWeight / float64(len(queryTerms))
	return math.Min(score, 1.0)
}

// scoreImportFrequency scores based on how often a file is imported
func (fr *FileRanker) scoreImportFrequency(filePath string, symbols []Symbol, allSymbols map[string][]Symbol) float64 {
	// Count how many other files reference symbols from this file
	fileName := filepath.Base(filePath)
	importCount := 0

	for otherFile, otherSymbols := range allSymbols {
		if otherFile == filePath {
			continue
		}

		// Check if any symbols reference this file
		for _, symbol := range otherSymbols {
			if strings.Contains(symbol.Signature, fileName) {
				importCount++
				break
			}
		}
	}

	// Normalize
	if importCount == 0 {
		return 0.0
	}

	return math.Min(float64(importCount)/10.0, 1.0)
}

// scoreDependencyDepth scores files that are central in the dependency graph
func (fr *FileRanker) scoreDependencyDepth(filePath string, allSymbols map[string][]Symbol) float64 {
	// Files with more exported symbols are likely more central
	symbols := allSymbols[filePath]
	exportedCount := 0

	for _, symbol := range symbols {
		// Check if symbol is exported (convention: starts with uppercase in Go, etc.)
		if len(symbol.Name) > 0 && isExported(symbol.Name) {
			exportedCount++
		}
	}

	if exportedCount == 0 {
		return 0.0
	}

	return math.Min(float64(exportedCount)/20.0, 1.0)
}

// scoreFileSize scores based on file size (prefer smaller files)
func (fr *FileRanker) scoreFileSize(filePath string) float64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0.5
	}

	size := info.Size()

	// Optimal size is around 200-500 lines (roughly 5KB-15KB)
	if size < 5000 {
		return 0.3
	} else if size < 15000 {
		return 1.0
	} else if size < 50000 {
		return 0.7
	} else {
		return 0.3
	}
}

// scoreSymbolDensity scores based on number of symbols per line
func (fr *FileRanker) scoreSymbolDensity(symbols []Symbol, filePath string) float64 {
	if len(symbols) == 0 {
		return 0.0
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return 0.0
	}

	// Approximate lines (1 line ~= 50 bytes)
	approxLines := info.Size() / 50
	if approxLines == 0 {
		return 0.0
	}

	density := float64(len(symbols)) / float64(approxLines)

	// Good density is around 1 symbol per 10-20 lines
	optimalDensity := 0.05 // 1/20
	score := 1.0 - math.Abs(density-optimalDensity)/optimalDensity

	return math.Max(0.0, math.Min(score, 1.0))
}

// tokenizeQuery splits a query into searchable terms
func (fr *FileRanker) tokenizeQuery(query string) []string {
	// Split by common delimiters
	query = strings.ToLower(query)
	replacer := strings.NewReplacer(
		"_", " ",
		"-", " ",
		".", " ",
		"/", " ",
		"\\", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		",", " ",
		";", " ",
		":", " ",
	)

	query = replacer.Replace(query)
	terms := strings.Fields(query)

	// Filter out common stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "the": true,
		"is": true, "are": true, "in": true, "on": true,
		"at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true,
	}

	filtered := make([]string, 0)
	for _, term := range terms {
		if len(term) > 1 && !stopWords[term] {
			filtered = append(filtered, term)
		}
	}

	return filtered
}

// tokenizeSymbolName splits a symbol name into terms (handles camelCase, snake_case, etc.)
func (fr *FileRanker) tokenizeSymbolName(name string) []string {
	terms := make([]string, 0)

	// Handle snake_case
	if strings.Contains(name, "_") {
		parts := strings.Split(name, "_")
		terms = append(terms, parts...)
	}

	// Handle kebab-case
	if strings.Contains(name, "-") {
		parts := strings.Split(name, "-")
		terms = append(terms, parts...)
	}

	// Handle camelCase and PascalCase
	var currentTerm strings.Builder
	for i, r := range name {
		if i > 0 && isUpperCase(r) && !isUpperCase(rune(name[i-1])) {
			if currentTerm.Len() > 0 {
				terms = append(terms, currentTerm.String())
				currentTerm.Reset()
			}
		}
		currentTerm.WriteRune(r)
	}
	if currentTerm.Len() > 0 {
		terms = append(terms, currentTerm.String())
	}

	// If no splitting occurred, just use the whole name
	if len(terms) == 0 {
		terms = append(terms, name)
	}

	return terms
}

// getSymbolTypeWeight returns importance weight for different symbol types
func (fr *FileRanker) getSymbolTypeWeight(symbolType SymbolType) float64 {
	weights := map[SymbolType]float64{
		SymbolTypeClass:     1.0,
		SymbolTypeInterface: 1.0,
		SymbolTypeStruct:    1.0,
		SymbolTypeTrait:     1.0,
		SymbolTypeFunction:  0.8,
		SymbolTypeMethod:    0.7,
		SymbolTypeEnum:      0.6,
		SymbolTypeModule:    0.5,
		SymbolTypeConstant:  0.3,
		SymbolTypeVariable:  0.2,
	}

	if weight, ok := weights[symbolType]; ok {
		return weight
	}

	return 0.5 // Default weight
}

// isExported checks if a symbol name is exported (public)
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}

	// Go convention: uppercase first letter
	return isUpperCase(rune(name[0]))
}

// isUpperCase checks if a rune is uppercase
func isUpperCase(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// RankByModificationTime ranks files by their modification time
func (fr *FileRanker) RankByModificationTime(files []string) []FileScore {
	scores := make([]FileScore, 0, len(files))

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		age := time.Since(info.ModTime())
		score := 1.0 / (1.0 + age.Hours()/24.0) // Decay over days

		scores = append(scores, FileScore{
			FilePath: file,
			Score:    score,
			Reasons:  []string{"modification time"},
		})
	}

	return scores
}

// RankBySymbolCount ranks files by their symbol count
func (fr *FileRanker) RankBySymbolCount(symbolsByFile map[string][]Symbol) []FileScore {
	scores := make([]FileScore, 0, len(symbolsByFile))

	for file, symbols := range symbolsByFile {
		score := math.Min(float64(len(symbols))/50.0, 1.0)

		scores = append(scores, FileScore{
			FilePath: file,
			Score:    score,
			Reasons:  []string{"symbol count"},
		})
	}

	return scores
}

// CombineScores combines multiple scoring strategies
func (fr *FileRanker) CombineScores(scoreSets [][]FileScore, weights []float64) []FileScore {
	if len(scoreSets) == 0 {
		return []FileScore{}
	}

	if len(weights) == 0 {
		// Equal weights
		weights = make([]float64, len(scoreSets))
		for i := range weights {
			weights[i] = 1.0 / float64(len(scoreSets))
		}
	}

	// Combine scores
	combined := make(map[string]float64)
	reasons := make(map[string][]string)

	for i, scoreSet := range scoreSets {
		for _, score := range scoreSet {
			combined[score.FilePath] += score.Score * weights[i]
			reasons[score.FilePath] = append(reasons[score.FilePath], score.Reasons...)
		}
	}

	// Convert to slice
	result := make([]FileScore, 0, len(combined))
	for file, score := range combined {
		result = append(result, FileScore{
			FilePath: file,
			Score:    score,
			Reasons:  reasons[file],
		})
	}

	return result
}
