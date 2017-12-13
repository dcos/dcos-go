package zkstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentValidate(t *testing.T) {
	type testCase struct {
		ident  Ident
		errMsg string
	}
	for _, test := range []testCase{
		{
			ident:  Ident{},
			errMsg: "invalid location: invalid name: cannot be blank",
		},
		{
			ident:  Ident{Location: Location{Name: "foo"}},
			errMsg: "invalid location: invalid category: cannot be blank",
		},
		{
			ident:  Ident{Location: Location{Name: "foo/bar"}},
			errMsg: "invalid location: invalid name: must match " + validNameRE.String(),
		},
		{
			ident:  Ident{Location: Location{Name: "foo", Category: "widgets"}, Version: "invalid/version"},
			errMsg: "invalid version: must match " + validNameRE.String(),
		},
		{
			ident: Ident{Location: Location{Name: "foo", Category: "widgets"}, Version: "my-version"},
		},
		{
			ident: Ident{Location: Location{Name: "foo", Category: "widgets/2017"}, Version: "my-version"},
		},
	} {
		err := test.ident.Validate()
		if errMsg(err) != test.errMsg {
			t.Fatalf("%v err:%v", test, err)
		}
	}
}

func TestActualZKVersion(t *testing.T) {
	require := require.New(t)
	ident := Ident{}
	require.EqualValues(-1, ident.actualZKVersion())
	ident.SetZKVersion(0)
	require.EqualValues(0, ident.actualZKVersion())
	ident.SetZKVersion(1)
	require.EqualValues(1, ident.actualZKVersion())
	ident.SetZKVersion(-1)
	require.EqualValues(-1, ident.actualZKVersion())
}
