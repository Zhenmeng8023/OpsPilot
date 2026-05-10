package agents

import (
	"database/sql"
	"testing"
)

func TestNormalizeBatchPlanInput(t *testing.T) {
	got := normalizeBatchPlanInput(BatchPlanInput{})
	if got.BatchSize != 50 || got.MaxBatches != 20 {
		t.Fatalf("unexpected defaults: %#v", got)
	}

	got = normalizeBatchPlanInput(BatchPlanInput{BatchSize: 1000, MaxBatches: 500})
	if got.BatchSize != 500 || got.MaxBatches != 200 {
		t.Fatalf("unexpected clamps: %#v", got)
	}
}

func TestFlattenDiagnosticPayload(t *testing.T) {
	raw := sql.NullString{String: `{"a":{"b":1},"list":[{"x":"y"}],"enabled":true}`, Valid: true}
	flat := flattenDiagnosticPayload(raw)
	if flat["a.b"] != "1" {
		t.Fatalf("expected a.b=1, got %q", flat["a.b"])
	}
	if flat["list[0].x"] != "y" {
		t.Fatalf("expected list[0].x=y, got %q", flat["list[0].x"])
	}
	if flat["enabled"] != "true" {
		t.Fatalf("expected enabled=true, got %q", flat["enabled"])
	}
}
