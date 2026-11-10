package service

import "testing"

func TestWithinCreditLimit(t *testing.T) {
	tests := []struct {
		taken, adding int
		want          bool
	}{
		{21, 3, true},
		{24, 3, false},
		{0, 24, true},
	}
	for _, test := range tests {
		if got := WithinCreditLimit(test.taken, test.adding); got != test.want {
			t.Errorf("WithinCreditLimit(%d, %d) = %t, want %t",
				test.taken, test.adding, got, test.want)
		}
	}
}

func TestCreditsAfterSwap(t *testing.T) {
	if got := CreditsAfterSwap(24, 3, 6); got != 27 {
		t.Fatalf("CreditsAfterSwap() = %d, want 27", got)
	}
}
