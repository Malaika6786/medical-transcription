package nhs

import "testing"

// 9434765919 is a well-known publicly documented valid test NHS number
// (modulus-11 checks out: see the calculation in this package's docs).
func TestValidateNHSNumber_Valid(t *testing.T) {
	cases := []string{"9434765919", "943 476 5919", "943-476-5919"}
	for _, c := range cases {
		normalized, err := ValidateNHSNumber(c)
		if err != nil {
			t.Errorf("ValidateNHSNumber(%q) unexpected error: %v", c, err)
		}
		if normalized != "9434765919" {
			t.Errorf("ValidateNHSNumber(%q) = %q, want 9434765919", c, normalized)
		}
	}
}

func TestValidateNHSNumber_Invalid(t *testing.T) {
	cases := []string{
		"",
		"12345",       // too short
		"12345678901", // too long
		"9434765918",  // wrong check digit (last digit changed)
		"abcdefghij",  // non-numeric
	}
	for _, c := range cases {
		if _, err := ValidateNHSNumber(c); err == nil {
			t.Errorf("ValidateNHSNumber(%q) expected an error, got none", c)
		}
	}
}

func TestFormatNHSNumber(t *testing.T) {
	if got := FormatNHSNumber("9434765919"); got != "943 476 5919" {
		t.Errorf("FormatNHSNumber = %q, want %q", got, "943 476 5919")
	}
}
