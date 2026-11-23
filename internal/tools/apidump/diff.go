package main

import (
	"fmt"
	"slices"
	"strings"
)

// contextLines is how many unchanged lines a diff shows around a change.
const contextLines = 2

// maxLCS bounds the table the diff fills in. Reports are a few thousand lines;
// past this size the changed middle is shown as removed and added whole.
const maxLCS = 4_000_000

type op struct {
	text string
	kind byte // ' ', '-' or '+'
}

// unifiedDiff shows what differs between two versions of a report: the lines
// that went (-) and the ones that came (+), each group with a little context.
// Reports are sorted and have one declaration per line, so a changed
// signature reads as the old line followed by the new one.
func unifiedDiff(name, old, updated string) string {
	ops := diffOps(splitLines(old), splitLines(updated))
	show := make([]bool, len(ops))
	for i, o := range ops {
		if o.kind == ' ' {
			continue
		}
		for j := max(0, i-contextLines); j <= min(len(ops)-1, i+contextLines); j++ {
			show[j] = true
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s (committed)\n+++ %s (from the code)\n", name, name)
	shown, skipped := false, false
	for i, o := range ops {
		if !show[i] {
			skipped = true

			continue
		}
		if skipped && shown {
			b.WriteString("...\n")
		}
		shown, skipped = true, false
		fmt.Fprintf(&b, "%c %s\n", o.kind, o.text)
	}

	return b.String()
}

func splitLines(s string) []string {
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// diffOps returns the edit script from a to b: the common prefix and suffix
// are set aside, and a longest common subsequence of the middle decides the
// rest.
func diffOps(a, b []string) []op {
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	midA, midB := a[prefix:len(a)-suffix], b[prefix:len(b)-suffix]
	ops := make([]op, 0, len(a)+len(b))
	for _, line := range a[:prefix] {
		ops = append(ops, op{line, ' '})
	}
	ops = append(ops, middleOps(midA, midB)...)
	for _, line := range a[len(a)-suffix:] {
		ops = append(ops, op{line, ' '})
	}

	return ops
}

func middleOps(a, b []string) []op {
	if len(a)*len(b) > maxLCS {
		ops := make([]op, 0, len(a)+len(b))
		for _, line := range a {
			ops = append(ops, op{line, '-'})
		}
		for _, line := range b {
			ops = append(ops, op{line, '+'})
		}

		return ops
	}
	// lcs[i][j] is the length of the longest common subsequence of a[i:] and b[j:].
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := range slices.Backward(a) {
		for j := range slices.Backward(b) {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var ops []op
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		next, nextI, nextJ := nextMiddleOp(a, b, lcs, i, j)
		ops = append(ops, next)
		i, j = nextI, nextJ
	}

	return ops
}

func nextMiddleOp(a, b []string, lcs [][]int, i, j int) (op, int, int) {
	switch {
	case i < len(a) && j < len(b) && a[i] == b[j]:
		return op{a[i], ' '}, i + 1, j + 1
	case j == len(b) || (i < len(a) && lcs[i+1][j] >= lcs[i][j+1]):
		return op{a[i], '-'}, i + 1, j
	default:
		return op{b[j], '+'}, i, j + 1
	}
}
