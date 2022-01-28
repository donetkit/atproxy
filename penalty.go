package atproxy

import (
	"sync"
	"time"
)

type GetPenalty func(dialer *Dialer, hostPort string) time.Duration

func (_ Def) Penalty() (
	onSelected OnSelected,
	onNotSelected OnNotSelected,
	get GetPenalty,
) {

	type key struct {
		dialer   *Dialer
		hostPort string
	}

	var l sync.Mutex
	infos := make(map[key][]bool)

	const threshold = 3

	set := func(dialer *Dialer, hostPort string, selected bool) {
		k := key{
			dialer:   dialer,
			hostPort: hostPort,
		}
		l.Lock()
		defer l.Unlock()
		infos[k] = append(infos[k], selected)
		if len(infos[k]) > threshold {
			copy(infos[k], infos[k][len(infos[k])-threshold:])
			infos[k] = infos[k][:threshold]
		}
	}

	onSelected = func(dialer *Dialer, hostPort string) {
		set(dialer, hostPort, true)
	}

	onNotSelected = func(dialer *Dialer, hostPort string) {
		set(dialer, hostPort, false)
	}

	get = func(dialer *Dialer, hostPort string) time.Duration {
		k := key{
			dialer:   dialer,
			hostPort: hostPort,
		}
		l.Lock()
		seq := infos[k]
		l.Unlock()
		n := 0
		for i := len(seq) - 1; i >= 0; i-- {
			if !seq[i] {
				n++
			} else {
				break
			}
		}
		if n >= threshold {
			return time.Millisecond * 200
		}
		return 0
	}

	return
}
