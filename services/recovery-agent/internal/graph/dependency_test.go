package graph_test

import (
	"testing"

	"github.com/antigravity/mono/services/recovery-agent/internal/graph"
	"github.com/antigravity/mono/services/recovery-agent/internal/models"
)

func makeActions() []models.UndoAction {
	return []models.UndoAction{
		{ActionID: "a1", ResourceID: "r1", Priority: 1, DependsOn: []string{}},
		{ActionID: "a2", ResourceID: "r2", Priority: 2, DependsOn: []string{"a1"}},
		{ActionID: "a3", ResourceID: "r3", Priority: 1, DependsOn: []string{}},
	}
}

func TestGraphBuildSuccess(t *testing.T) {
	_, err := graph.Build(makeActions())
	if err != nil {
		t.Fatalf("graph.Build failed: %v", err)
	}
}

func TestTopologicalOrderRespectsDependencies(t *testing.T) {
	g, _ := graph.Build(makeActions())
	ordered, err := g.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder failed: %v", err)
	}
	posA1, posA2 := -1, -1
	for i, a := range ordered {
		if a.ActionID == "a1" { posA1 = i }
		if a.ActionID == "a2" { posA2 = i }
	}
	if posA1 < 0 || posA2 < 0 {
		t.Fatal("a1 or a2 missing from order")
	}
	if posA1 >= posA2 {
		t.Errorf("a1 (pos=%d) should precede a2 (pos=%d)", posA1, posA2)
	}
}

func TestCycleDetection(t *testing.T) {
	cyclic := []models.UndoAction{
		{ActionID: "x", ResourceID: "r1", Priority: 1, DependsOn: []string{"y"}},
		{ActionID: "y", ResourceID: "r2", Priority: 1, DependsOn: []string{"x"}},
	}
	g, err := graph.Build(cyclic)
	if err != nil {
		t.Fatalf("Build should not fail for cyclic graph: %v", err)
	}
	_, err = g.TopologicalOrder()
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestEmptyGraph(t *testing.T) {
	g, err := graph.Build([]models.UndoAction{})
	if err != nil {
		t.Fatalf("empty Build failed: %v", err)
	}
	ordered, err := g.TopologicalOrder()
	if err != nil {
		t.Fatalf("empty TopologicalOrder failed: %v", err)
	}
	if len(ordered) != 0 {
		t.Fatalf("expected 0 actions, got %d", len(ordered))
	}
}
