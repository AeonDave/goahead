//go:build ignore

//go:ahead functions

package variadic

import "strings"

// joinAll demonstrates variadic function support - can accept any number of arguments
func joinAll(sep string, parts ...string) string {
	return strings.Join(parts, sep)
}

// concat joins strings without separator
func concat(parts ...string) string {
	return strings.Join(parts, "")
}

// sum adds all numbers together
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// maxOf returns the maximum value
func maxOf(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	max := nums[0]
	for _, n := range nums[1:] {
		if n > max {
			max = n
		}
	}
	return max
}
