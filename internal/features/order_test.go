package features

import "testing"

func TestOrderDependsOnAndInstallsAfter(t *testing.T) {
	assignments := []string{"a", "b", "c"}
	metas := map[string]Metadata{
		"b": {DependsOn: map[string]map[string]interface{}{"a": {}}},
		"c": {InstallsAfter: []string{"b"}},
	}
	out, err := Order(assignments, metas)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}

	pos := map[string]int{}
	for i, id := range out {
		pos[id] = i
	}
	if !(pos["a"] < pos["b"] && pos["b"] < pos["c"]) {
		t.Fatalf("unexpected order: %#v", out)
	}
}

func TestOrderCycle(t *testing.T) {
	assignments := []string{"a", "b"}
	metas := map[string]Metadata{
		"a": {DependsOn: map[string]map[string]interface{}{"b": {}}},
		"b": {DependsOn: map[string]map[string]interface{}{"a": {}}},
	}
	if _, err := Order(assignments, metas); err == nil {
		t.Fatalf("expected cycle error")
	}
}
