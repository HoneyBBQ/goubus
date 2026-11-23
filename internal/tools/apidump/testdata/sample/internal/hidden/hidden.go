// Package hidden stands in for an internal package whose types must not
// appear in a public signature.
package hidden

// Secret is a type a caller cannot name.
type Secret struct {
	// Value is promoted through an alias.
	Value int
}

// Reveal is a method promoted through an alias.
func (s *Secret) Reveal() int { return s.Value }
