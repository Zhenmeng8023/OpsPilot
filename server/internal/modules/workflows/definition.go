package workflows

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	defaultFailurePolicy = "stop_on_failure"
	maxWorkflowNodes     = 50
)

type Definition struct {
	Nodes         []Node `json:"nodes"`
	Edges         []Edge `json:"edges"`
	MaxParallel   int    `json:"maxParallel,omitempty"`
	FailurePolicy string `json:"failurePolicy,omitempty"`
}

type Node struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	Name           string                 `json:"name,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	TimeoutSeconds uint                   `json:"timeoutSeconds,omitempty"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func normalizeAndValidateDefinition(raw string) (Definition, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Definition{}, "", errors.New("workflow definition is required")
	}
	var def Definition
	if err := json.Unmarshal([]byte(raw), &def); err != nil {
		return Definition{}, "", errors.New("workflow definition must be valid JSON")
	}
	if len(def.Nodes) == 0 {
		return Definition{}, "", errors.New("workflow must contain at least one node")
	}
	if len(def.Nodes) > maxWorkflowNodes {
		return Definition{}, "", fmt.Errorf("workflow cannot contain more than %d nodes", maxWorkflowNodes)
	}
	if def.MaxParallel <= 0 {
		def.MaxParallel = 1
	}
	if def.MaxParallel > 20 {
		return Definition{}, "", errors.New("maxParallel cannot be greater than 20")
	}
	def.FailurePolicy = strings.TrimSpace(def.FailurePolicy)
	if def.FailurePolicy == "" {
		def.FailurePolicy = defaultFailurePolicy
	}
	if def.FailurePolicy != "stop_on_failure" && def.FailurePolicy != "stop_workflow" && def.FailurePolicy != "skip_downstream" && def.FailurePolicy != "continue" {
		return Definition{}, "", errors.New("failurePolicy must be stop_on_failure, stop_workflow, skip_downstream, or continue")
	}

	nodes := make(map[string]Node, len(def.Nodes))
	graph := make(map[string][]string, len(def.Nodes))
	for index := range def.Nodes {
		node := &def.Nodes[index]
		node.ID = strings.TrimSpace(node.ID)
		node.Type = strings.TrimSpace(node.Type)
		node.Name = strings.TrimSpace(node.Name)
		if node.ID == "" {
			return Definition{}, "", errors.New("node id is required")
		}
		if len(node.ID) > 64 {
			return Definition{}, "", errors.New("node id cannot be longer than 64 characters")
		}
		if _, exists := nodes[node.ID]; exists {
			return Definition{}, "", fmt.Errorf("duplicate node id: %s", node.ID)
		}
		if !validNodeType(node.Type) {
			return Definition{}, "", fmt.Errorf("unsupported node type: %s", node.Type)
		}
		if node.TimeoutSeconds > maxWaitSeconds {
			return Definition{}, "", fmt.Errorf("node %s timeoutSeconds cannot exceed %d", node.ID, maxWaitSeconds)
		}
		if err := validateNodeConfig(*node); err != nil {
			return Definition{}, "", err
		}
		nodes[node.ID] = *node
		graph[node.ID] = []string{}
	}
	for index := range def.Edges {
		edge := &def.Edges[index]
		edge.From = strings.TrimSpace(edge.From)
		edge.To = strings.TrimSpace(edge.To)
		if edge.From == "" || edge.To == "" {
			return Definition{}, "", errors.New("edge from and to are required")
		}
		if edge.From == edge.To {
			return Definition{}, "", errors.New("edge cannot point to the same node")
		}
		if _, ok := nodes[edge.From]; !ok {
			return Definition{}, "", fmt.Errorf("edge references unknown from node: %s", edge.From)
		}
		if _, ok := nodes[edge.To]; !ok {
			return Definition{}, "", fmt.Errorf("edge references unknown to node: %s", edge.To)
		}
		graph[edge.From] = append(graph[edge.From], edge.To)
	}
	if err := rejectIsolatedNodes(def); err != nil {
		return Definition{}, "", err
	}
	if hasCycle(graph) {
		return Definition{}, "", errors.New("workflow graph cannot contain cycles")
	}
	normalized, err := json.Marshal(def)
	if err != nil {
		return Definition{}, "", err
	}
	return def, string(normalized), nil
}

func rejectIsolatedNodes(def Definition) error {
	if len(def.Nodes) <= 1 {
		return nil
	}
	degree := make(map[string]int, len(def.Nodes))
	for _, node := range def.Nodes {
		degree[node.ID] = 0
	}
	for _, edge := range def.Edges {
		degree[edge.From]++
		degree[edge.To]++
	}
	for _, node := range def.Nodes {
		if degree[node.ID] == 0 {
			return fmt.Errorf("node %s is isolated", node.ID)
		}
	}
	return nil
}

func validateNodeConfig(node Node) error {
	switch node.Type {
	case "task":
		if nodeConfigString(node, "taskId") == "" {
			return fmt.Errorf("task node %s requires config.taskId", node.ID)
		}
	case "condition":
		if err := validateConditionNode(node); err != nil {
			return err
		}
	case "approval":
		if err := validateApprovalNode(node); err != nil {
			return err
		}
	case "notification":
		if nodeConfigString(node, "title") == "" && node.Name == "" {
			return fmt.Errorf("notification node %s requires name or config.title", node.ID)
		}
	case "webhook", "webhook-call":
		if err := validateWebhookCallNode(node); err != nil {
			return err
		}
	case "wait":
		if err := validateWaitNode(node); err != nil {
			return err
		}
	}
	return nil
}

func validNodeType(value string) bool {
	switch value {
	case "task", "condition", "approval", "notification", "webhook", "webhook-call", "wait", "incident":
		return true
	default:
		return false
	}
}

func hasCycle(graph map[string][]string) bool {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(node string) bool {
		if visiting[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visiting[node] = true
		for _, next := range graph[node] {
			if visit(next) {
				return true
			}
		}
		visiting[node] = false
		visited[node] = true
		return false
	}
	for node := range graph {
		if visit(node) {
			return true
		}
	}
	return false
}
