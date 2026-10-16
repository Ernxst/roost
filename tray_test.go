package main

import "testing"

func TestByline(t *testing.T) {
	for _, c := range []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"ada"}, "From @ada"},
		{[]string{"ada", "grace"}, "From @ada and @grace"},
		{[]string{"ada", "grace", "alan"}, "From @ada, @grace and @alan"},
		{[]string{"ada", "grace", "alan", "kay"}, "From @ada, @grace, @alan and 1 other"},
		{[]string{"ada", "grace", "alan", "kay", "joan"}, "From @ada, @grace, @alan and 2 others"},
	} {
		if got := byline(c.in); got != c.want {
			t.Errorf("byline(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
