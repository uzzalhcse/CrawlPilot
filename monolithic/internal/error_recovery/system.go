package error_recovery

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/internal/logger"
	"go.uber.org/zap"
)

// ErrorRecoverySystem orchestrates all error recovery components
type ErrorRecoverySystem struct {
	analyzer       *ErrorPatternAnalyzer
	rulesEngine    *ContextAwareRulesEngine
	aiEngine       *AIReasoningEngine
	learningEngine *LearningEngine
	enabled        bool
}

// Config for the error recovery system
type SystemConfig struct {
	Enabled        bool
	AnalyzerConfig AnalyzerConfig
	MinSuccessRate float64
	MinUsageCount  int
	AIEnabled      bool
}

// NewErrorRecoverySystem creates a new error recovery system
func NewErrorRecoverySystem(config SystemConfig, rules []ContextAwareRule, aiClient AIClient) *ErrorRecoverySystem {
	learningEngine := NewLearningEngine(config.MinSuccessRate, config.MinUsageCount)

	return &ErrorRecoverySystem{
		analyzer:       NewErrorPatternAnalyzer(config.AnalyzerConfig),
		rulesEngine:    NewContextAwareRulesEngine(rules),
		aiEngine:       NewAIReasoningEngine(aiClient, learningEngine, config.AIEnabled),
		learningEngine: learningEngine,
		enabled:        config.Enabled,
	}
}

// GetAnalyzer returns the error pattern analyzer
func (s *ErrorRecoverySystem) GetAnalyzer() *ErrorPatternAnalyzer {
	return s.analyzer
}

// HandleError processes an error through the recovery system
func (s *ErrorRecoverySystem) HandleError(ctx context.Context, err error, execCtx *ExecutionContext) (*Solution, error) {
	logger.Info("🔍 Error Recovery System: Analyzing error",
		zap.String("error", err.Error()),
		zap.String("url", execCtx.URL),
		zap.String("domain", execCtx.Domain),
		zap.Int("status_code", execCtx.Response.StatusCode))

	if !s.enabled {
		logger.Warn("❌ Error Recovery System is disabled")
		return nil, fmt.Errorf("error recovery system is disabled")
	}

	// Step 1: Check if recovery should be activated
	logger.Debug("📊 Checking if pattern analyzer should activate recovery...")
	decision := s.analyzer.ShouldActivate(err, execCtx)
	if !decision.ShouldActivate {
		logger.Debug("⏭️  Error recovery not activated - threshold not met",
			zap.String("reason", decision.Reason))
		return nil, fmt.Errorf("recovery not needed")
	}

	logger.Info("✅ Pattern detected - activating recovery",
		zap.String("reason", decision.Reason),
		zap.String("pattern_type", decision.ErrorPattern.Type),
		zap.Float64("error_rate", decision.ErrorPattern.ErrorRate),
		zap.Int("consecutive_errors", decision.ErrorPattern.ConsecutiveCount))

	// Step 2: Try to find a matching rule
	logger.Debug("🔎 Searching for matching rule...")
	solution, err := s.rulesEngine.FindSolution(err, execCtx)
	if err == nil {
		logger.Info("✅ Rule-based solution found",
			zap.String("rule", solution.RuleName),
			zap.Float64("confidence", solution.Confidence),
			zap.Int("actions_count", len(solution.Actions)),
			zap.String("solution_type", "rule"))
		return solution, nil
	}

	logger.Debug("❌ No matching rule found", zap.Error(err))

	// Step 3: Fall back to AI reasoning if enabled
	if s.aiEngine != nil {
		logger.Info("🤖 No rule matched - falling back to AI reasoning...")
		aiSolution, aiErr := s.aiEngine.ReasonAndSolve(ctx, execCtx, err)
		if aiErr == nil {
			logger.Info("✅ AI solution generated",
				zap.Float64("confidence", aiSolution.Confidence),
				zap.Int("actions_count", len(aiSolution.Actions)),
				zap.String("solution_type", "ai"))
			return aiSolution, nil
		}
		logger.Warn("❌ AI reasoning failed", zap.Error(aiErr))
	} else {
		logger.Debug("⚠️  AI reasoning disabled, no fallback available")
	}

	logger.Warn("❌ No solution found for error", zap.String("error", err.Error()))
	return nil, fmt.Errorf("no solution found")
}

// ApplySolution applies a solution's actions
func (s *ErrorRecoverySystem) ApplySolution(solution *Solution, execCtx *ExecutionContext) error {
	logger.Info("Applying solution",
		zap.String("rule", solution.RuleName),
		zap.Int("action_count", len(solution.Actions)))

	for _, action := range solution.Actions {
		if err := s.applyAction(action, execCtx); err != nil {
			logger.Error("Failed to apply action",
				zap.String("action", action.Type),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// applyAction applies a single action
func (s *ErrorRecoverySystem) applyAction(action Action, execCtx *ExecutionContext) error {
	logger.Debug("Applying action",
		zap.String("type", action.Type),
		zap.Any("parameters", action.Parameters))

	// Actions are applied by the caller based on the action type
	// This method is a placeholder for validation
	switch action.Type {
	case "enable_stealth", "rotate_proxy", "adjust_timeout",
		"reduce_workers", "add_delay", "wait", "pause_execution", "resume_execution":
		return nil
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// TrackSuccess tracks a successful solution
func (s *ErrorRecoverySystem) TrackSuccess(solution *Solution, execCtx *ExecutionContext) {
	if s.learningEngine != nil {
		s.learningEngine.TrackSuccess(solution, execCtx)
	}
}

// TrackFailure tracks a failed solution
func (s *ErrorRecoverySystem) TrackFailure(solution *Solution) {
	if s.learningEngine != nil {
		s.learningEngine.TrackFailure(solution)
	}
}

// AddRule adds a new rule to the engine
func (s *ErrorRecoverySystem) AddRule(rule ContextAwareRule) {
	s.rulesEngine.AddRule(rule)
}

// UpdateRule updates an existing rule
func (s *ErrorRecoverySystem) UpdateRule(rule ContextAwareRule) error {
	return s.rulesEngine.UpdateRule(rule)
}

// RemoveRule removes a rule by ID
func (s *ErrorRecoverySystem) RemoveRule(id string) error {
	return s.rulesEngine.RemoveRule(id)
}

// LoadRules loads rules from the database and updates the engine
func (s *ErrorRecoverySystem) LoadRules(rules []ContextAwareRule) {
	s.rulesEngine = NewContextAwareRulesEngine(rules)
}
