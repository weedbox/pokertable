package util

import (
	"math/rand"
	"time"
)

/*
	RotateIntArray 給定 source, 以 startIndex 當作第一個元素做 Rotations
		- @param source Given source array
		- @param startIndex Base index for the rotation
		- @return rotated source
	Example:
		- Given: []int{0, 1, 2, 3, 4}, startIndex = 2
		- Output: []int{2, 3, 4, 0, 1}
*/
func RotateIntArray(source []int, startIndex int) []int {
	if startIndex > len(source) {
		startIndex = startIndex % len(source)
	}
	return append(source[startIndex:], source[:startIndex]...)
}

func RandomInt(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}
