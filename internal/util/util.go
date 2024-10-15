/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
			if labelRegex.MatchString(k) {
				out[k] = v
			}
		}
		return
	}
	return labelMatcher, err
}
