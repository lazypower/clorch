package tui

import (
	"testing"

	"github.com/lazypower/clorch/internal/state"
)

func TestBuildTree_NoLineage(t *testing.T) {
	agents := []state.AgentState{
		{SessionID: "a", ProjectName: "proj-a"},
		{SessionID: "b", ProjectName: "proj-b"},
		{SessionID: "c", ProjectName: "proj-c"},
	}
	tree := buildTree(agents)
	if len(tree) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(tree))
	}
	for _, e := range tree {
		if e.prefix != "" {
			t.Errorf("root agent %s should have empty prefix, got %q", e.agent.SessionID, e.prefix)
		}
	}
	// Order preserved
	if tree[0].agent.SessionID != "a" || tree[1].agent.SessionID != "b" || tree[2].agent.SessionID != "c" {
		t.Error("expected original order preserved for roots")
	}
}

func TestBuildTree_SingleChild(t *testing.T) {
	agents := []state.AgentState{
		{SessionID: "parent", ProjectName: "proj"},
		{SessionID: "child", ProjectName: "branch", BranchedFrom: "parent"},
	}
	tree := buildTree(agents)
	if len(tree) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tree))
	}
	if tree[0].agent.SessionID != "parent" {
		t.Errorf("expected parent first, got %s", tree[0].agent.SessionID)
	}
	if tree[0].prefix != "" {
		t.Errorf("parent should have empty prefix, got %q", tree[0].prefix)
	}
	if tree[1].agent.SessionID != "child" {
		t.Errorf("expected child second, got %s", tree[1].agent.SessionID)
	}
	if tree[1].prefix != "└── " {
		t.Errorf("single child should have └── prefix, got %q", tree[1].prefix)
	}
}

func TestBuildTree_MultipleChildren(t *testing.T) {
	agents := []state.AgentState{
		{SessionID: "parent", ProjectName: "proj"},
		{SessionID: "child1", ProjectName: "b1", BranchedFrom: "parent"},
		{SessionID: "child2", ProjectName: "b2", BranchedFrom: "parent"},
	}
	tree := buildTree(agents)
	if len(tree) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(tree))
	}
	if tree[1].prefix != "├── " {
		t.Errorf("first of multiple children should have ├── prefix, got %q", tree[1].prefix)
	}
	if tree[2].prefix != "└── " {
		t.Errorf("last child should have └── prefix, got %q", tree[2].prefix)
	}
}

func TestBuildTree_DeepNesting(t *testing.T) {
	agents := []state.AgentState{
		{SessionID: "root", ProjectName: "proj"},
		{SessionID: "child", ProjectName: "b1", BranchedFrom: "root"},
		{SessionID: "grandchild", ProjectName: "b2", BranchedFrom: "child"},
	}
	tree := buildTree(agents)
	if len(tree) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(tree))
	}
	if tree[0].agent.SessionID != "root" {
		t.Errorf("expected root first")
	}
	if tree[1].agent.SessionID != "child" {
		t.Errorf("expected child second")
	}
	if tree[2].agent.SessionID != "grandchild" {
		t.Errorf("expected grandchild third")
	}
	// Grandchild should have deeper indent
	if tree[2].prefix != "    └── " {
		t.Errorf("grandchild should have indented └── prefix, got %q", tree[2].prefix)
	}
}

func TestBuildTree_OrphanChild(t *testing.T) {
	// Child references a parent that's not in the list — treated as root
	agents := []state.AgentState{
		{SessionID: "a", ProjectName: "proj-a"},
		{SessionID: "orphan", ProjectName: "proj-orphan", BranchedFrom: "gone"},
	}
	tree := buildTree(agents)
	if len(tree) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(tree))
	}
	for _, e := range tree {
		if e.prefix != "" {
			t.Errorf("orphan agent %s should be treated as root with empty prefix, got %q", e.agent.SessionID, e.prefix)
		}
	}
}

func TestBuildTree_MixedRootsAndBranches(t *testing.T) {
	agents := []state.AgentState{
		{SessionID: "standalone", ProjectName: "solo"},
		{SessionID: "parent", ProjectName: "proj"},
		{SessionID: "child", ProjectName: "branch", BranchedFrom: "parent"},
		{SessionID: "another", ProjectName: "other"},
	}
	tree := buildTree(agents)
	if len(tree) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(tree))
	}
	// standalone is root, then parent+child grouped, then another
	if tree[0].agent.SessionID != "standalone" {
		t.Errorf("expected standalone first, got %s", tree[0].agent.SessionID)
	}
	if tree[1].agent.SessionID != "parent" {
		t.Errorf("expected parent second, got %s", tree[1].agent.SessionID)
	}
	if tree[2].agent.SessionID != "child" {
		t.Errorf("expected child third, got %s", tree[2].agent.SessionID)
	}
	if tree[3].agent.SessionID != "another" {
		t.Errorf("expected another fourth, got %s", tree[3].agent.SessionID)
	}
}
