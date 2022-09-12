package util

import "math/rand"

func FindValInSlice(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func ReverseSlice(slice []string) []string {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}

func TakeRandServerHost(serverLists []string) string {
	if len(serverLists) == 0 {
		return ""
	}
	i := rand.Intn(len(serverLists))
	if i == len(serverLists) && i != 0 {
		i = i - 1
	}
	return serverLists[i]
}

func Contains(str_array []string, target string) bool {
	for _, element := range str_array {
		if target == element {
			return true
		}
	}
	return false
}
