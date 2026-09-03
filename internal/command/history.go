package command

type History struct {
	items []string
	index int
	draft string
}

func (h *History) Add(value string) {
	if value == "" {
		return
	}
	if len(h.items) == 0 || h.items[len(h.items)-1] != value {
		h.items = append(h.items, value)
	}
	h.index, h.draft = len(h.items), ""
}

func (h *History) Previous(current string) string {
	if len(h.items) == 0 {
		return current
	}
	if h.index == len(h.items) {
		h.draft = current
	}
	if h.index > 0 {
		h.index--
	}
	return h.items[h.index]
}

func (h *History) Next() string {
	if len(h.items) == 0 {
		return ""
	}
	if h.index < len(h.items)-1 {
		h.index++
		return h.items[h.index]
	}
	h.index = len(h.items)
	return h.draft
}

func (h *History) Items() []string { return append([]string(nil), h.items...) }
