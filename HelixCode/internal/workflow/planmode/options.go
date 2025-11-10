package planmode

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"
)

// PlanOption represents an implementation option
type PlanOption struct {
	ID          string
	Title       string
	Description string
	Plan        *Plan
	Pros        []string
	Cons        []string
	Rank        int
	Score       float64
	Recommended bool
}

// Selection represents a user's option selection
type Selection struct {
	OptionID  string
	Timestamp time.Time
	Feedback  string
	Custom    bool // If user provided custom modifications
}

// Comparison contains a comparison of options
type Comparison struct {
	Options  []*PlanOption
	Criteria []string
	Matrix   [][]ComparisonCell
	Summary  string
}

// ComparisonCell represents a single comparison cell
type ComparisonCell struct {
	OptionID  string
	Criterion string
	Value     string
	Score     float64
}

// RankCriterion defines criteria for ranking options
type RankCriterion struct {
	Name   string
	Weight float64
	Type   CriterionType
}

// CriterionType defines the type of ranking criterion
type CriterionType int

const (
	CriterionSpeed CriterionType = iota
	CriterionSafety
	CriterionSimplicity
	CriterionMaintainability
	CriterionPerformance
	CriterionCost
)

// String returns string representation of criterion type
func (ct CriterionType) String() string {
	return [...]string{"Speed", "Safety", "Simplicity", "Maintainability", "Performance", "Cost"}[ct]
}

// RankedOption is an option with ranking information
type RankedOption struct {
	Option *PlanOption
	Rank   int
	Score  float64
	Scores map[string]float64 // Scores per criterion
}

// OptionPresenter presents plan options to the user
type OptionPresenter interface {
	// Present presents options to the user
	Present(ctx context.Context, options []*PlanOption) (*Selection, error)

	// CompareOptions compares multiple options
	CompareOptions(options []*PlanOption) (*Comparison, error)

	// RankOptions ranks options by various criteria
	RankOptions(options []*PlanOption, criteria []RankCriterion) ([]*RankedOption, error)
}

// CLIOptionPresenter presents options via CLI
type CLIOptionPresenter struct {
	output io.Writer
	input  io.Reader
}

// NewCLIOptionPresenter creates a new CLI option presenter
func NewCLIOptionPresenter(output io.Writer, input io.Reader) OptionPresenter {
	return &CLIOptionPresenter{
		output: output,
		input:  input,
	}
}

// Present presents options to the user
func (p *CLIOptionPresenter) Present(ctx context.Context, options []*PlanOption) (*Selection, error) {
	fmt.Fprintln(p.output, "\n=== Implementation Options ===")

	for i, opt := range options {
		fmt.Fprintf(p.output, "Option %d: %s", i+1, opt.Title)
		if opt.Recommended {
			fmt.Fprint(p.output, " [RECOMMENDED]")
		}
		fmt.Fprintln(p.output)
		fmt.Fprintf(p.output, "Score: %.1f/100\n", opt.Score)
		fmt.Fprintf(p.output, "Description: %s\n", opt.Description)
		fmt.Fprintln(p.output)

		fmt.Fprintln(p.output, "Pros:")
		for _, pro := range opt.Pros {
			fmt.Fprintf(p.output, "  + %s\n", pro)
		}
		fmt.Fprintln(p.output)

		fmt.Fprintln(p.output, "Cons:")
		for _, con := range opt.Cons {
			fmt.Fprintf(p.output, "  - %s\n", con)
		}
		fmt.Fprintln(p.output)

		fmt.Fprintf(p.output, "Estimated Duration: %s\n", opt.Plan.Estimates.Duration)
		fmt.Fprintf(p.output, "Complexity: %s\n", opt.Plan.Estimates.Complexity)
		fmt.Fprintf(p.output, "Confidence: %.0f%%\n", opt.Plan.Estimates.Confidence*100)
		fmt.Fprintln(p.output, "\n---")
	}

	// Prompt for selection
	fmt.Fprintf(p.output, "Select an option (1-%d): ", len(options))

	var choice int
	_, err := fmt.Fscanln(p.input, &choice)
	if err != nil {
		return nil, fmt.Errorf("failed to read selection: %w", err)
	}

	if choice < 1 || choice > len(options) {
		return nil, fmt.Errorf("invalid choice: %d", choice)
	}

	selected := options[choice-1]

	return &Selection{
		OptionID:  selected.ID,
		Timestamp: time.Now(),
	}, nil
}

// CompareOptions compares multiple options
func (p *CLIOptionPresenter) CompareOptions(options []*PlanOption) (*Comparison, error) {
	criteria := []string{
		"Complexity",
		"Duration",
		"Confidence",
		"Risk Level",
		"Score",
	}

	matrix := make([][]ComparisonCell, len(options))
	for i, opt := range options {
		matrix[i] = make([]ComparisonCell, len(criteria))

		matrix[i][0] = ComparisonCell{
			OptionID:  opt.ID,
			Criterion: "Complexity",
			Value:     opt.Plan.Estimates.Complexity.String(),
		}

		matrix[i][1] = ComparisonCell{
			OptionID:  opt.ID,
			Criterion: "Duration",
			Value:     opt.Plan.Estimates.Duration.String(),
		}

		matrix[i][2] = ComparisonCell{
			OptionID:  opt.ID,
			Criterion: "Confidence",
			Value:     fmt.Sprintf("%.0f%%", opt.Plan.Estimates.Confidence*100),
			Score:     opt.Plan.Estimates.Confidence,
		}

		riskLevel := calculateRiskLevel(opt.Plan.Risks)
		matrix[i][3] = ComparisonCell{
			OptionID:  opt.ID,
			Criterion: "Risk Level",
			Value:     riskLevel,
		}

		matrix[i][4] = ComparisonCell{
			OptionID:  opt.ID,
			Criterion: "Score",
			Value:     fmt.Sprintf("%.1f", opt.Score),
			Score:     opt.Score,
		}
	}

	return &Comparison{
		Options:  options,
		Criteria: criteria,
		Matrix:   matrix,
	}, nil
}

// RankOptions ranks options by various criteria
func (p *CLIOptionPresenter) RankOptions(options []*PlanOption, criteria []RankCriterion) ([]*RankedOption, error) {
	ranked := make([]*RankedOption, len(options))

	for i, opt := range options {
		rankedOpt := &RankedOption{
			Option: opt,
			Scores: make(map[string]float64),
		}

		totalScore := 0.0
		totalWeight := 0.0

		for _, criterion := range criteria {
			score := scoreByCriterion(opt, criterion.Type)
			rankedOpt.Scores[criterion.Name] = score
			totalScore += score * criterion.Weight
			totalWeight += criterion.Weight
		}

		if totalWeight > 0 {
			rankedOpt.Score = totalScore / totalWeight
		}

		ranked[i] = rankedOpt
	}

	// Sort by score (descending)
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	// Assign ranks
	for i, rankedOpt := range ranked {
		rankedOpt.Rank = i + 1
	}

	return ranked, nil
}

// calculateRiskLevel calculates overall risk level
func calculateRiskLevel(risks []Risk) string {
	if len(risks) == 0 {
		return "Low"
	}

	maxImpact := ImpactLow
	for _, risk := range risks {
		if risk.Impact > maxImpact && risk.Likelihood >= LikelihoodMedium {
			maxImpact = risk.Impact
		}
	}

	switch maxImpact {
	case ImpactCritical:
		return "Critical"
	case ImpactHigh:
		return "High"
	case ImpactMedium:
		return "Medium"
	default:
		return "Low"
	}
}

// scoreByCriterion scores an option by a specific criterion
func scoreByCriterion(opt *PlanOption, criterionType CriterionType) float64 {
	switch criterionType {
	case CriterionSpeed:
		// Lower duration is better
		duration := opt.Plan.Estimates.Duration.Hours()
		if duration < 1 {
			return 100.0
		} else if duration < 4 {
			return 80.0
		} else if duration < 8 {
			return 60.0
		} else if duration < 24 {
			return 40.0
		}
		return 20.0

	case CriterionSafety:
		// Lower risk is better
		riskScore := 100.0
		for _, risk := range opt.Plan.Risks {
			impact := float64(risk.Impact+1) * 10
			likelihood := float64(risk.Likelihood+1) * 10
			riskScore -= (impact * likelihood) / 10
		}
		if riskScore < 0 {
			riskScore = 0
		}
		return riskScore

	case CriterionSimplicity:
		// Lower complexity is better
		complexityScore := map[Complexity]float64{
			ComplexityLow:      100.0,
			ComplexityMedium:   75.0,
			ComplexityHigh:     50.0,
			ComplexityVeryHigh: 25.0,
		}
		return complexityScore[opt.Plan.Estimates.Complexity]

	case CriterionMaintainability:
		// Higher confidence and lower complexity
		return (opt.Plan.Estimates.Confidence * 50) + (scoreByCriterion(opt, CriterionSimplicity) * 0.5)

	case CriterionPerformance:
		// Based on confidence and fewer steps
		stepScore := 100.0
		if len(opt.Plan.Steps) > 10 {
			stepScore = 50.0
		} else if len(opt.Plan.Steps) > 5 {
			stepScore = 75.0
		}
		return (opt.Plan.Estimates.Confidence * 50) + (stepScore * 0.5)

	case CriterionCost:
		// Combination of duration and complexity
		return (scoreByCriterion(opt, CriterionSpeed) * 0.5) + (scoreByCriterion(opt, CriterionSimplicity) * 0.5)

	default:
		return opt.Score
	}
}
