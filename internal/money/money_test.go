package money

import "testing"

func TestParseCents(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{in: "100", want: 10000},
		{in: "100.00", want: 10000},
		{in: "100.5", want: 10050},
		{in: "0", want: 0},
		{in: "0.01", want: 1},
		{in: "999999999999999.99", want: 99999999999999999},
		{in: "", wantErr: true},
		{in: "1.234", wantErr: true},
		{in: "abc", wantErr: true},
		{in: "-5", wantErr: true},
		{in: "1,000", wantErr: true},
		{in: "1.", wantErr: true},
		{in: ".5", wantErr: true},
		{in: "99999999999999999999", wantErr: true},
	}
	for _, c := range cases {
		got, err := ParseCents(c.in)
		if c.wantErr {
			if err == nil {
				t.Fatalf("ParseCents(%q) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseCents(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("ParseCents(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestFloatCents(t *testing.T) {
	cases := []struct {
		in   float64
		want int64
	}{
		{in: 100.00, want: 10000},
		{in: 100.55, want: 10055},
		{in: 0.5, want: 50},
		{in: 0.005, want: 1},
		{in: 0, want: 0},
		{in: -5, want: 0},
	}
	for _, c := range cases {
		if got := FloatCents(c.in); got != c.want {
			t.Fatalf("FloatCents(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
