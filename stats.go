package atproxy

import "github.com/reusee/dscope"

type OnSelected func(
	dialer *Dialer,
	hostPort string,
)

var _ dscope.Reducer = OnSelected(nil)

func (OnSelected) IsReducer() {}

func (Def) OnSelected() OnSelected {
	return func(
		dialer *Dialer,
		hostPort string,
	) {
	}
}

type OnNotSelected func(
	dialer *Dialer,
	hostPort string,
)

var _ dscope.Reducer = OnNotSelected(nil)

func (OnNotSelected) IsReducer() {}

func (Def) OnNotSelected() OnNotSelected {
	return func(
		dialer *Dialer,
		hostPort string,
	) {
	}
}
