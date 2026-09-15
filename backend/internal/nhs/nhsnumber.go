// Package nhs contains the NHS/SystmOne integration surface added per
// SYSTMONE_INTEGRATION_REPORT.md: NHS number validation, a PDS lookup
// client, a minimal FHIR message builder for GP Connect: Send Document, and
// a MESH transport client. None of this can be exercised against the real
// NHS Spine without NHS-issued credentials (a registered supplier
// organisation, MESH mailbox, PDS access, and — where user-restricted APIs
// are involved — a CIS2 client registration); see docs/nhs/README.md for
// exactly what's real code today vs. what still needs those credentials
// wired in before it can talk to a live or NHS test environment.
package nhs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var digitsOnly = regexp.MustCompile(`\D`)

// NormalizeNHSNumber strips spaces/hyphens from a user-entered NHS number
// ("485 777 3456" or "485-777-3456" -> "4857773456").
func NormalizeNHSNumber(raw string) string {
	return digitsOnly.ReplaceAllString(raw, "")
}

// ValidateNHSNumber checks that raw (after normalization) is a
// structurally valid NHS number: exactly 10 digits, whose modulus-11 check
// digit (the published NHS algorithm — see docs/nhs/README.md for the
// source) matches the 10th digit. This confirms the number is
// well-formed; it does NOT confirm the number belongs to any real
// patient — only a live PDS trace (LookupByNHSNumber) can do that.
func ValidateNHSNumber(raw string) (normalized string, err error) {
	n := NormalizeNHSNumber(raw)
	if len(n) != 10 {
		return "", fmt.Errorf("NHS number must be exactly 10 digits, got %d", len(n))
	}
	sum := 0
	for i := 0; i < 9; i++ {
		d, convErr := strconv.Atoi(string(n[i]))
		if convErr != nil {
			return "", fmt.Errorf("NHS number must be numeric: %w", convErr)
		}
		// Weights 10..2 for digits 1..9, per the NHS modulus-11 algorithm.
		sum += d * (10 - i)
	}
	remainder := sum % 11
	checkDigit := 11 - remainder
	switch checkDigit {
	case 11:
		checkDigit = 0
	case 10:
		return "", fmt.Errorf("NHS number %q fails the modulus-11 check (invalid number)", n)
	}
	actual, convErr := strconv.Atoi(string(n[9]))
	if convErr != nil {
		return "", fmt.Errorf("NHS number must be numeric: %w", convErr)
	}
	if actual != checkDigit {
		return "", fmt.Errorf("NHS number %q fails the modulus-11 check digit (expected %d, got %d)", n, checkDigit, actual)
	}
	return n, nil
}

// FormatNHSNumber renders a normalized 10-digit NHS number as
// "123 456 7890", the standard display grouping.
func FormatNHSNumber(normalized string) string {
	if len(normalized) != 10 {
		return normalized
	}
	return strings.Join([]string{normalized[0:3], normalized[3:6], normalized[6:10]}, " ")
}
