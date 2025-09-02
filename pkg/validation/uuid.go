package validation

import (
	"regexp"
	"strings"
)

// ValidateUUID checks if a string is a valid UUID format
func ValidateUUID(uuid string) bool {
	// UUID regex pattern - either all dashes or no dashes
	uuidWithDashes := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	uuidWithoutDashes := `^[0-9a-fA-F]{32}$`

	matched, err := regexp.MatchString(uuidWithDashes, uuid)
	if err != nil {
		return false
	}
	if matched {
		return true
	}

	matched, err = regexp.MatchString(uuidWithoutDashes, uuid)
	if err != nil {
		return false
	}
	return matched
}

// NormalizeUUID converts a UUID to lowercase with dashes
func NormalizeUUID(uuid string) string {
	// Remove all dashes first
	clean := strings.ReplaceAll(uuid, "-", "")
	clean = strings.ToLower(clean)

	// Add dashes in the correct positions
	if len(clean) == 32 {
		return clean[:8] + "-" + clean[8:12] + "-" + clean[12:16] + "-" + clean[16:20] + "-" + clean[20:]
	}

	return uuid // Return original if not 32 characters
}

// IsValidVersion checks if a version array is valid
func IsValidVersion(version [3]int) bool {
	for _, v := range version {
		if v < 0 {
			return false
		}
	}
	return true
}

// CompareVersions compares two version arrays
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 [3]int) int {
	for i := 0; i < 3; i++ {
		if v1[i] < v2[i] {
			return -1
		}
		if v1[i] > v2[i] {
			return 1
		}
	}
	return 0
}
