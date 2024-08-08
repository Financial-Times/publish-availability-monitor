package config

import (
	"reflect"
	"testing"
)

func TestGetXReadPolicies(t *testing.T) {
	tests := []struct {
		name                 string
		enabledPublications  []Publication
		expectedReadPolicies []string
	}{
		{
			name: "single publication",
			enabledPublications: []Publication{
				{Name: "Publication1", UUID: "UUID1"},
			},
			expectedReadPolicies: []string{"PBLC_READ_UUID1"},
		},
		{
			name: "multiple publications",
			enabledPublications: []Publication{
				{Name: "Publication1", UUID: "UUID1"},
				{Name: "Publication2", UUID: "UUID2"},
			},
			expectedReadPolicies: []string{"PBLC_READ_UUID1", "PBLC_READ_UUID2"},
		},
		{
			name:                 "no publications",
			enabledPublications:  []Publication{},
			expectedReadPolicies: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &PublicationsConfig{
				EnabledPublications: tt.enabledPublications,
			}
			got := cfg.GetXReadPolicies()
			if !reflect.DeepEqual(got, tt.expectedReadPolicies) {
				t.Errorf("GetXReadPolicies() = %v, want %v", got, tt.expectedReadPolicies)
			}
		})
	}
}
