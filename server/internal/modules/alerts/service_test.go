package alerts

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		operator  string
		threshold float64
		want      bool
	}{
		{"gt", 2, ">", 1, true},
		{"gte equal", 2, ">=", 2, true},
		{"lt false", 2, "<", 1, false},
		{"neq", 2, "!=", 1, true},
		{"unknown", 2, "contains", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compare(tt.value, tt.operator, tt.threshold); got != tt.want {
				t.Fatalf("compare() = %v, want %v", got, tt.want)
			}
		})
	}
}
