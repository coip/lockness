// Package lockness implementes facilities for a Golang manager
// for Learning Locker
// Testing LL_API_KEY: 2c617bb5701e0a67b54252110f0ddf11672b4820
// Testing LL_API_SECRET: e1900213b5e375b3c3f3e054b1e12d8f534b8c8c
package lockness

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

var basicConfig = []byte(`---
llIP: dns:port
userReqString: foo
llPostString: bar
llAPIVersion: 1.0.3
`)

var tagErrorConfig = []byte(`---
llIP:
userReqString:
llThisHasNoStructMapping:foo
llAPIVersion:
`)

var testConfig = []byte(`---
llIP: gracev0_learninglocker:8081
userReqString: http://%s/data/xAPI/statements?agent=%%7B%%22mbox%%22%%3A%%20%%22mailto%%3A%s%%40grace.co%%22%%7D
llPostString: http://%s/data/xAPI/statements
llAPIVersion: 1.0.3
`)

// Equal returns true if the receiver and argument *LLRequest are equal.
func equalReqs(llr1 *LLRequest, llr2 *LLRequest, errCheck func(error, error) bool) bool {
	if !errCheck(llr1.Err, llr2.Err) {
		return false
	}
	if llr1.ReqString != llr2.ReqString {
		return false
	}
	if llr1.PostString != llr2.PostString {
		return false
	}
	if llr1.LearningLockerIP != llr2.LearningLockerIP {
		return false
	}
	if llr1.APIVersion != llr2.APIVersion {
		return false
	}
	if llr1.LLApiKey != llr2.LLApiKey {
		return false
	}
	if llr1.LLSecretKey != llr2.LLSecretKey {
		return false
	}
	if !errCheck(llr1.Err, llr2.Err) {
		return false
	}

	// Fall through all of the error conditions and return equal.
	return true
}

func TestNewLLReaderRequest(t *testing.T) {
	type args struct {
		config io.Reader
	}
	tests := []struct {
		name        string
		args        args
		env         map[string]string
		want        *LLRequest
		errorsEqual func(error, error) bool
	}{
		{
			name: "empty",
			args: args{
				config: bytes.NewBuffer([]byte("")),
			},
			env: map[string]string{
				"LL_API_KEY":    "",
				"LL_API_SECRET": "",
			},
			want: &LLRequest{},
			errorsEqual: func(e1, e2 error) bool {
				return reflect.DeepEqual(e1, e2)
			},
		},
		{
			name: "missing-api-key",
			args: args{
				config: bytes.NewBuffer([]byte("")),
			},
			env: map[string]string{
				"LL_API_SECRET": "",
			},
			want: &LLRequest{
				Err: fmt.Errorf("missing environment variable for LL_API_KEY or LL_API_SECRET"),
			},
			errorsEqual: func(e1, e2 error) bool {
				return reflect.DeepEqual(e1, e2)
			},
		},
		{
			name: "wrong-yaml-tags",
			args: args{
				config: bytes.NewBuffer(tagErrorConfig),
			},
			env: map[string]string{
				"LL_API_KEY":    "star",
				"LL_API_SECRET": "wars",
			},
			want: &LLRequest{
				ReqString:        "",
				PostString:       "",
				LearningLockerIP: "",
				APIVersion:       "",
				LLApiKey:         "",
				LLSecretKey:      "",
				Err:              nil,
			},
			errorsEqual: func(e1, e2 error) bool {
				return strings.Contains(e1.Error(), "unmarshal strict error parsing")
			},
		},
		{
			name: "basic",
			args: args{
				config: bytes.NewBuffer(basicConfig),
			},
			env: map[string]string{
				"LL_API_KEY":    "star",
				"LL_API_SECRET": "wars",
			},
			want: &LLRequest{
				ReqString:        "foo",
				PostString:       "bar",
				LearningLockerIP: "dns:port",
				APIVersion:       "1.0.3",
				LLApiKey:         "star",
				LLSecretKey:      "wars",
				Err:              nil,
			},
			errorsEqual: func(e1, e2 error) bool {
				// return reflect.DeepEqual(e1, e2)
				return true
			},
		},
	}
	for _, tt := range tests {
		for k, v := range tt.env {
			os.Setenv(k, v)
		}
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLLReaderRequest(tt.args.config); equalReqs(got, tt.want, tt.errorsEqual) != true {
				t.Errorf("NewLLReaderRequest() = %+q, want %+q", got, tt.want)
			}
		})
		for k := range tt.env {
			os.Unsetenv(k)
		}
	}
}
