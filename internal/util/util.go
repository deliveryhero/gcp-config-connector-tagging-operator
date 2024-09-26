package util

import (
	"regexp"
)

func LimitLabelsWithRegex(targetLabels string) (func(map[string]string) map[string]string, error) {
	labelRegex, err := regexp.Compile(targetLabels)
	if err != nil {
		return nil, err
	}

	labelMatcher := func(in map[string]string) (out map[string]string) {
		out = make(map[string]string, len(in))
		for k, v := range in {
			if labelRegex.MatchString(v) {
				out[k] = v
			}
		}
		return
	}
	return labelMatcher, err
}
