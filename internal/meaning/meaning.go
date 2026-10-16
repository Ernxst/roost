// Package meaning turns text into vectors that lie close together when the
// texts mean similar things, using the sentence embedding macOS ships with,
// so nothing leaves the Mac.
package meaning

import "math"

// Cosine is the cosine similarity of two vectors of the same length.
func Cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}
