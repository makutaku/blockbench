package validation

import (
	"testing"
)

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		expected bool
	}{
		// Valid UUIDs
		{"valid UUID with dashes", "12345678-1234-1234-1234-123456789abc", true},
		{"valid UUID without dashes", "123456781234123412341234567890ab", true},
		{"valid UUID mixed case", "12345678-1234-1234-1234-123456789ABC", true},
		{"valid UUID all uppercase", "12345678-1234-1234-1234-123456789ABC", true},
		{"valid UUID all lowercase", "12345678-1234-1234-1234-123456789abc", true},

		// Invalid UUIDs
		{"too short", "12345678-1234-1234-1234-123456789ab", false},
		{"too long", "12345678-1234-1234-1234-123456789abcd", false},
		{"invalid characters", "12345678-1234-1234-1234-123456789abg", false},
		{"empty string", "", false},
		{"partial dashes", "12345678-12341234-1234-123456789abc", false},
		{"wrong dash positions", "123456781-234-1234-1234-123456789abc", false},
		{"invalid format", "not-a-uuid-at-all", false},
		{"numbers only but wrong length", "12345678123412341234123456789012345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateUUID(tt.uuid)
			if result != tt.expected {
				t.Errorf("ValidateUUID(%q) = %v, want %v", tt.uuid, result, tt.expected)
			}
		})
	}
}

func TestNormalizeUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UUID with dashes",
			input:    "12345678-1234-1234-1234-123456789ABC",
			expected: "12345678-1234-1234-1234-123456789abc",
		},
		{
			name:     "UUID without dashes",
			input:    "123456781234123412341234567890AB",
			expected: "12345678-1234-1234-1234-1234567890ab",
		},
		{
			name:     "Mixed case with dashes",
			input:    "AbCdEf01-2345-6789-AbCd-Ef0123456789",
			expected: "abcdef01-2345-6789-abcd-ef0123456789",
		},
		{
			name:     "Already normalized",
			input:    "12345678-1234-1234-1234-123456789abc",
			expected: "12345678-1234-1234-1234-123456789abc",
		},
		{
			name:     "Invalid length - returns original",
			input:    "too-short",
			expected: "too-short",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeUUID(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeUUID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  [3]int
		expected bool
	}{
		{"valid version 1.0.0", [3]int{1, 0, 0}, true},
		{"valid version 1.2.3", [3]int{1, 2, 3}, true},
		{"valid version 0.0.0", [3]int{0, 0, 0}, true},
		{"valid version with large numbers", [3]int{999, 888, 777}, true},
		{"invalid version with negative major", [3]int{-1, 0, 0}, false},
		{"invalid version with negative minor", [3]int{1, -1, 0}, false},
		{"invalid version with negative patch", [3]int{1, 0, -1}, false},
		{"invalid version all negative", [3]int{-1, -2, -3}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidVersion(tt.version)
			if result != tt.expected {
				t.Errorf("IsValidVersion(%v) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       [3]int
		v2       [3]int
		expected int
	}{
		// Equal versions
		{"equal versions", [3]int{1, 0, 0}, [3]int{1, 0, 0}, 0},
		{"equal complex versions", [3]int{2, 3, 4}, [3]int{2, 3, 4}, 0},
		{"equal zero versions", [3]int{0, 0, 0}, [3]int{0, 0, 0}, 0},

		// v1 greater than v2
		{"major version higher", [3]int{2, 0, 0}, [3]int{1, 0, 0}, 1},
		{"minor version higher", [3]int{1, 2, 0}, [3]int{1, 1, 0}, 1},
		{"patch version higher", [3]int{1, 0, 2}, [3]int{1, 0, 1}, 1},
		{"complex v1 > v2", [3]int{2, 1, 3}, [3]int{2, 1, 2}, 1},

		// v1 less than v2
		{"major version lower", [3]int{1, 0, 0}, [3]int{2, 0, 0}, -1},
		{"minor version lower", [3]int{1, 1, 0}, [3]int{1, 2, 0}, -1},
		{"patch version lower", [3]int{1, 0, 1}, [3]int{1, 0, 2}, -1},
		{"complex v1 < v2", [3]int{1, 9, 9}, [3]int{2, 0, 0}, -1},

		// Edge cases
		{"major difference overrides minor/patch", [3]int{1, 9, 9}, [3]int{2, 0, 0}, -1},
		{"minor difference overrides patch", [3]int{1, 1, 9}, [3]int{1, 2, 0}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%v, %v) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateUUID(b *testing.B) {
	uuid := "12345678-1234-1234-1234-123456789abc"
	for i := 0; i < b.N; i++ {
		ValidateUUID(uuid)
	}
}

func BenchmarkNormalizeUUID(b *testing.B) {
	uuid := "12345678-1234-1234-1234-123456789ABC"
	for i := 0; i < b.N; i++ {
		NormalizeUUID(uuid)
	}
}

func BenchmarkCompareVersions(b *testing.B) {
	v1 := [3]int{1, 2, 3}
	v2 := [3]int{1, 2, 4}
	for i := 0; i < b.N; i++ {
		CompareVersions(v1, v2)
	}
}
