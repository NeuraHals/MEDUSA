package graph

import (
	"fmt"

	"github.com/antigravity/mono/services/recovery-agent/internal/models"
)

// DependencyGraph builds a topological execution order for undo actions
// respecting the DependsOn constraints declared in each UndoAction.
// The AOA emits LIFO-ordered actions; this graph layer adds a secondary
// dependency-aware pass to handle multi-resource dependency chains.
type DependencyGraph struct {
	actions map[string]*models.UndoAction
	deps    map[string][]string // action_id -> list of action_ids it depends on
}

// Build constructs the graph from a slice of undo actions.
func Build(actions []models.UndoAction) (*DependencyGraph, error) {
	g := &DependencyGraph{
		actions: make(map[string]*models.UndoAction, len(actions)),
		deps:    make(map[string][]string, len(actions)),
	}
	for i := range actions {
		a := actions[i]
		g.actions[a.ActionID] = &a
		g.deps[a.ActionID] = a.DependsOn
	}
	// Validate: all declared dependencies must exist in the manifest
	for id, deps := range g.deps {
		for _, dep := range deps {
			if _, ok := g.actions[dep]; !ok {
				return nil, fmt.Errorf("action %s depends on unknown action %s", id, dep)
			}
		}
	}
	return g, nil
}

// TopologicalOrder returns undo actions in a valid dependency-respecting execution order.
// Uses Kahn's BFS algorithm: nodes with zero unmet prerequisites execute first.
// If DependsOn is empty for all actions, execution is priority-ordered.
func (g *DependencyGraph) TopologicalOrder() ([]models.UndoAction, error) {
	// inDegree[id] = number of prerequisites not yet satisfied for action id
	inDegree := make(map[string]int, len(g.actions))
	for id := range g.actions {
		inDegree[id] = 0
	}
	// Build reverse adjacency: after[prereq] = list of nodes that depend on prereq
	after := make(map[string][]string, len(g.actions))
	for id, deps := range g.deps {
		inDegree[id] += len(deps)
		for _, prereq := range deps {
			after[prereq] = append(after[prereq], id)
		}
	}

	// Seed queue with nodes that have no prerequisites
	queue := make([]string, 0, len(g.actions))
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]models.UndoAction, 0, len(g.actions))
	for len(queue) > 0 {
		// Pick the highest-priority (lowest Priority value) ready action
		best := 0
		for i := 1; i < len(queue); i++ {
			if g.actions[queue[i]].Priority < g.actions[queue[best]].Priority {
				best = i
			}
		}
		id := queue[best]
		queue = append(queue[:best], queue[best+1:]...)
		result = append(result, *g.actions[id])

		// Unlock nodes that depended on this one
		for _, dependent := range after[id] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(g.actions) {
		return nil, fmt.Errorf("cycle detected in rollback dependency graph")
	}

	return result, nil
}

