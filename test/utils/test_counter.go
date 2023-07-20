package utils

import "fmt"

var counter = 1

func GetNextCounter() string {
	counter++
	return fmt.Sprint(counter)
}
