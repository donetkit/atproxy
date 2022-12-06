package atproxy

import (
	"sync"
	"time"
)

type GetPenalty func(dialer *Dialer, hostPort string) time.Duration

func (Def) Penalty() (
	onSelected OnSelected,
	onNotSelected OnNotSelected,
	get GetPenalty,
) {

	type Info struct {
		window   time.Time
		selected []*Dialer
	}

	var l sync.Mutex
	infos := make(map[string]*Info)

	const threshold = 3

	set := func(dialer *Dialer, hostPort string) {
		l.Lock()
		defer l.Unlock()
		info := infos[hostPort]
		if info == nil {
			info = &Info{
				window: time.Now(),
			}
			infos[hostPort] = info
		}
		if time.Since(info.window) > time.Hour {
			info.window = time.Now()
			info.selected = info.selected[:0]
		}
		info.selected = append(info.selected, dialer)
		i := 0
		for len(info.selected) > threshold {
			info.selected[i] = info.selected[len(info.selected)-1]
			i++
			info.selected = info.selected[:len(info.selected)-1]
		}
	}

	onSelected = func(dialer *Dialer, hostPort string) {
		set(dialer, hostPort)
	}

	onNotSelected = func(dialer *Dialer, hostPort string) {
	}

	get = func(dialer *Dialer, hostPort string) time.Duration {
		l.Lock()
		defer l.Unlock()
		info := infos[hostPort]
		if info == nil {
			return 0
		}
		numNotSelected := 0
		for _, d := range info.selected {
			if d != dialer {
				numNotSelected++
			}
		}
		return time.Millisecond * 200 * time.Duration(numNotSelected)
	}

	return
}
