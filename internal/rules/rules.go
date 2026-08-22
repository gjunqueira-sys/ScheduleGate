package rules

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gjunqueira-sys/ScheduleGate/internal/model"
	"gopkg.in/yaml.v3"
)

// Rule defines a single pattern rule to check against schedule tasks.
type Rule struct {
	Name        string            `yaml:"name"`
	Match       map[string]string `yaml:"match"`       // field -> glob pattern
	Constraints Constraints       `yaml:"constraints"` // numeric constraints
	MinCount    int               `yaml:"min_count"`
	MaxCount    int               `yaml:"max_count"` // 0 = no max
}

// Constraints for numeric field validations.
type Constraints struct {
	MinDuration float64 `yaml:"min_duration"`
	MaxDuration float64 `yaml:"max_duration"` // 0 = no max
}

// RuleSet is a collection of rules loaded from YAML.
type RuleSet struct {
	Rules []Rule `yaml:"rules"`
}

// RuleResult is the evaluation result for a single rule.
type RuleResult struct {
	Rule          Rule
	MatchingTasks []*model.Task
	Count         int
	Passing       bool
	Message       string
}

// LoadRules loads rules from a YAML file.
func LoadRules(filePath string) (*RuleSet, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var ruleSet RuleSet
	if err := yaml.Unmarshal(data, &ruleSet); err != nil {
		return nil, err
	}

	return &ruleSet, nil
}

// filterWorkTasks returns only non-summary, non-milestone detail tasks.
// Summary (rollup) tasks and milestones carry no independent work content and
// must not influence pattern-rule count or constraint evaluations.
//
// Signature: filterWorkTasks(tasks []*model.Task) []*model.Task
// Arguments:
//   - tasks: full task slice from a parsed Schedule
//
// Returns: new slice containing only work/detail tasks
func filterWorkTasks(tasks []*model.Task) []*model.Task {
	result := make([]*model.Task, 0, len(tasks))
	for _, t := range tasks {
		if !t.IsSummary && !t.IsMilestone {
			result = append(result, t)
		}
	}
	return result
}

// Evaluate evaluates all rules against a schedule.
// Summary (rollup) tasks are excluded before rule evaluation because they
// aggregate child values, carry no real work content, and have no independent
// predecessor/successor logic links.
//
// Signature: (rs *RuleSet) Evaluate(schedule *model.Schedule) []RuleResult
// Arguments:
//   - schedule: parsed Schedule whose Tasks will be filtered to work tasks only
//
// Returns: slice of RuleResult, one per rule in the set
func (rs *RuleSet) Evaluate(schedule *model.Schedule) []RuleResult {
	workTasks := filterWorkTasks(schedule.Tasks)
	results := make([]RuleResult, 0, len(rs.Rules))

	for _, rule := range rs.Rules {
		result := evaluateRule(rule, workTasks)
		results = append(results, result)
	}

	return results
}

func evaluateRule(rule Rule, tasks []*model.Task) RuleResult {
	var matching []*model.Task
	var constraintExcluded []*model.Task

	for _, task := range tasks {
		if matchesRule(rule, task) {
			matching = append(matching, task)
			if !satisfiesConstraints(rule.Constraints, task) {
				constraintExcluded = append(constraintExcluded, task)
			}
		}
	}

	effectiveCount := len(matching) - len(constraintExcluded)
	passing := true
	var messages []string

	if len(constraintExcluded) > 0 {
		messages = append(messages, fmt.Sprintf("%d tasks excluded by duration constraints", len(constraintExcluded)))
	}

	if rule.MinCount > 0 && effectiveCount < rule.MinCount {
		passing = false
		messages = append(messages, "below minimum count")
	}

	if rule.MaxCount > 0 && effectiveCount > rule.MaxCount {
		passing = false
		messages = append(messages, "exceeds maximum count")
	}

	message := "OK"
	if len(messages) > 0 {
		message = strings.Join(messages, ", ")
	}

	return RuleResult{
		Rule:          rule,
		MatchingTasks: matching,
		Count:         effectiveCount,
		Passing:       passing,
		Message:       message,
	}
}

func matchesRule(rule Rule, task *model.Task) bool {
	if len(rule.Match) == 0 {
		return false
	}
	for field, pattern := range rule.Match {
		fieldValue := getFieldValue(task, field)
		matched, _ := path.Match(strings.ToLower(pattern), strings.ToLower(fieldValue))
		if !matched {
			return false
		}
	}
	return true
}

func satisfiesConstraints(constraints Constraints, task *model.Task) bool {
	if constraints.MinDuration > 0 && task.Duration < constraints.MinDuration {
		return false
	}
	if constraints.MaxDuration > 0 && task.Duration > constraints.MaxDuration {
		return false
	}
	return true
}

func getFieldValue(task *model.Task, field string) string {
	switch strings.ToLower(field) {
	case "name", "task_name":
		return task.Name
	case "wbs", "wbs_code":
		return task.WBS
	case "id", "task_id":
		return task.TaskID
	case "resources", "resource_names":
		return task.Resources
	case "predecessors":
		return task.Predecessors
	case "constraint_type":
		return task.ConstraintType
	case "discipline", "task_discipline":
		return task.Discipline
	case "mechanical_segment_nbr", "mech_segment", "mechanical_segment":
		return task.MechanicalSegmentNbr
	case "control_segment_nbr", "control_segment", "controls_segment":
		return task.ControlSegmentNbr
	default:
		return ""
	}
}
