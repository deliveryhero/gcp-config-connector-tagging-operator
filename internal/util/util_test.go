package util

import (
	"regexp"
	"testing"
)

func TestLimitLabelsWithRegex(t *testing.T) {
	type args struct {
		targetLabels string
	}
	tests := []struct {
		name    string
		args    args
		want    func(map[string]string) map[string]string
		wantErr bool
	}{
		{
			name: "Valid regex",
			args: args{
				targetLabels: ".*",
			},
			want: func(in map[string]string) (out map[string]string) {
				out = make(map[string]string, len(in))
				for k, v := range in {
					if regexp.MustCompile(".*").MatchString(v) {
						out[k] = v
					}
				}
				return
			},
			wantErr: false,
		},
		{
			name: "Regex matching specific label value",
			args: args{
				targetLabels: "value-.*",
			},
			want: func(in map[string]string) (out map[string]string) {
				out = make(map[string]string, len(in))
				for k, v := range in {
					if regexp.MustCompile("value-.*").MatchString(v) {
						out[k] = v
					}
				}
				return
			},
			wantErr: false,
		},
		{
			name: "Invalid regex",
			args: args{
				targetLabels: "*(",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LimitLabelsWithRegex(tt.args.targetLabels)
			if (err != nil) != tt.wantErr {
				t.Errorf("LimitLabelsWithRegex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			testLabels := map[string]string{
				"label1": "value-1",
				"label2": "value-2",
				"label3": "different-value",
			}

			expected := tt.want(testLabels)
			actual := got(testLabels)

			var fail bool
			for k, v := range expected {
				if found, ok := actual[k]; !ok {
					t.Errorf("label '%s' expected at location '%s', but not available", v, k)
					fail = true
				} else if found != v {
					t.Errorf("label '%s' expected at location '%s', but found '%s'", v, k, found)
					fail = true
				}
			}
			for k, v := range actual {
				if _, ok := expected[k]; !ok {
					t.Errorf("label '%s' expected at location '%s', but not available", v, k)
					fail = true
				}
				// Do not flag mismatches again
			}
			if fail {
				t.Fail()
			}
		})
	}
}
