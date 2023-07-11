package crawl

import "fmt"

type TestEvents []string

func (ev *TestEvents) Reset() {
	*ev = (*ev)[:0]
}

func (ev *TestEvents) Add(a ...any) {
	s := fmt.Sprint(a...)
	*ev = append(*ev, s)
}

func (ev *TestEvents) Addf(format string, a ...any) {
	s := fmt.Sprintf(format, a...)
	*ev = append(*ev, s)
}
