package budget

import "fmt"

type Limits struct {
	MaxWallTimeSeconds int
	MaxRetries         int
	MaxStepCount       int
	MaxToolCalls       int
	MaxEstimatedCost   *float64
	MaxTokens          *int
}

type Usage struct {
	WallTimeSeconds int
	RetryCount      int
	StepCount       int
	ToolCallCount   int
	EstimatedCost   *float64
	Tokens          *int
}

type Result struct {
	Exceeded   bool
	Violations []string
}

func Evaluate(limits Limits, usage Usage) Result {
	violations := make([]string, 0, 6)

	if limits.MaxWallTimeSeconds > 0 && usage.WallTimeSeconds > limits.MaxWallTimeSeconds {
		violations = append(violations, fmt.Sprintf("wall_time_seconds>%d", limits.MaxWallTimeSeconds))
	}
	if limits.MaxRetries > 0 && usage.RetryCount > limits.MaxRetries {
		violations = append(violations, fmt.Sprintf("retry_count>%d", limits.MaxRetries))
	}
	if limits.MaxStepCount > 0 && usage.StepCount > limits.MaxStepCount {
		violations = append(violations, fmt.Sprintf("step_count>%d", limits.MaxStepCount))
	}
	if limits.MaxToolCalls > 0 && usage.ToolCallCount > limits.MaxToolCalls {
		violations = append(violations, fmt.Sprintf("tool_call_count>%d", limits.MaxToolCalls))
	}
	if limits.MaxEstimatedCost != nil && usage.EstimatedCost != nil && *usage.EstimatedCost > *limits.MaxEstimatedCost {
		violations = append(violations, fmt.Sprintf("estimated_cost>%.4f", *limits.MaxEstimatedCost))
	}
	if limits.MaxTokens != nil && usage.Tokens != nil && *usage.Tokens > *limits.MaxTokens {
		violations = append(violations, fmt.Sprintf("tokens>%d", *limits.MaxTokens))
	}

	return Result{
		Exceeded:   len(violations) > 0,
		Violations: violations,
	}
}
