package budget

import "testing"

func TestEvaluateExceeded(t *testing.T) {
	t.Parallel()

	costLimit := 10.0
	tokenLimit := 500
	cost := 11.5
	tokens := 700
	result := Evaluate(
		Limits{
			MaxWallTimeSeconds: 300,
			MaxRetries:         2,
			MaxStepCount:       20,
			MaxToolCalls:       40,
			MaxEstimatedCost:   &costLimit,
			MaxTokens:          &tokenLimit,
		},
		Usage{
			WallTimeSeconds: 301,
			RetryCount:      3,
			StepCount:       21,
			ToolCallCount:   41,
			EstimatedCost:   &cost,
			Tokens:          &tokens,
		},
	)

	if !result.Exceeded {
		t.Fatal("expected exceeded")
	}
	if len(result.Violations) != 6 {
		t.Fatalf("expected 6 violations, got %d (%v)", len(result.Violations), result.Violations)
	}
}

func TestEvaluateWithinLimits(t *testing.T) {
	t.Parallel()

	result := Evaluate(
		Limits{MaxWallTimeSeconds: 300, MaxRetries: 2, MaxStepCount: 20, MaxToolCalls: 40},
		Usage{WallTimeSeconds: 60, RetryCount: 1, StepCount: 10, ToolCallCount: 20},
	)
	if result.Exceeded {
		t.Fatalf("expected not exceeded, got %v", result.Violations)
	}
}
