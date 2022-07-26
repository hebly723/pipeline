package pipeline

import (
	"testing"
)

func Test_extractParams(t *testing.T) {
	type args struct {
		result []byte
		e      ExtractParams
		params map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "first test",
			args: args{
				result: []byte(
					"one|for|all",
				),
				e: ExtractParams{
					Type:      "split",
					Separator: "|",
					Result: []ExtractResult{
						{
							Index:        0,
							Name:         "one",
							Assert:       "one",
							AssertResult: false,
						},
					},
				},
				params: map[string]string{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := extractParams(tt.args.result, tt.args.e, tt.args.params); (err != nil) != tt.wantErr {
				t.Errorf("extractParams() error = %v, wantErr %v, %+v\n", err, tt.wantErr, tt.args.e)
			}
		})
	}
}
