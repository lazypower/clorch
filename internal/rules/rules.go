package rules

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Action int

const (
	Ask     Action = iota
	Approve
	Deny
)

type Rule struct {
	Tools   []string `yaml:"tools"`
	Pattern string   `yaml:"pattern,omitempty"`
	Action  string   `yaml:"action"`
}

type Config struct {
	YOLO  bool   `yaml:"yolo"`
	Rules []Rule `yaml:"rules"`
}

type Engine struct {
	config Config
}

func NewEngine(path string) (*Engine, error) {
	e := &Engine{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return e, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, &e.config); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) Evaluate(toolName string, summary string) Action {
	for _, rule := range e.config.Rules {
		if !toolMatches(rule.Tools, toolName) {
			continue
		}
		if rule.Pattern != "" && !strings.Contains(summary, rule.Pattern) {
			continue
		}
		switch rule.Action {
		case "approve":
			return Approve
		case "deny":
			return Deny
		}
	}
	if e.config.YOLO {
		return Approve
	}
	return Ask
}

func (e *Engine) IsYOLO() bool        { return e.config.YOLO }
func (e *Engine) SetYOLO(enabled bool) { e.config.YOLO = enabled }

func toolMatches(tools []string, name string) bool {
	if len(tools) == 0 {
		return true
	}
	for _, t := range tools {
		if t == name {
			return true
		}
	}
	return false
}
