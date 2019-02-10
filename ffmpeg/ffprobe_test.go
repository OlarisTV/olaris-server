package ffmpeg

import (
	_ "bytes"
	"math/big"
	"reflect"
	"testing"
)

func Test_parseRational(t *testing.T) {
	type args struct {
		r string
	}
	tests := []struct {
		name    string
		args    args
		want    *big.Rat
		wantErr bool
	}{
		{
			"success",
			args{"2/5000"},
			big.NewRat(2, 5000),
			false,
		}, {
			"garbage",
			args{"garbage"},
			&big.Rat{},
			true,
		}, {
			"twoSlashes",
			args{"2/5000/3000"},
			&big.Rat{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRational(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRational() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRational() = %v, want %v", got, tt.want)
			}
		})
	}
}
