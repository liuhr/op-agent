package util

import "regexp"

//Analyze whether the string has the keyword x1...xn
func HasSuffixWithKeyWork(analyzeString string,pattern string) (bool, error) {
	if matched, err := regexp.MatchString(pattern,analyzeString); err != nil {
		return false,err
	} else {
		if matched {
			return true,nil
		}
	}
	return false, nil
}

func IsIP(ip string) (b bool) {
	m, _ := regexp.MatchString("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}$", ip)

	if  !m {
		return false
	}
	return true
}