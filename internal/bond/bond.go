package bond

import (
	"fmt"
	"strconv"
	"strings"
)

type Rec struct {
	Title, Body string
	Tags        []string
}

func Sample() Rec {
	return Rec{Title: "tc-north-12", Body: "section=up feed=12", Tags: []string{"tc-north"}}
}

func Seed() []Rec {
	return []Rec{
		Sample(),
		{Title: "tc-north-12-b", Body: "section=dn feed=12", Tags: []string{"tc-north"}},
	}
}

func AfterWrite(getMin func() (string, error), setMin func(string) error, body string) error {
	c, err := ParseFeed(body)
	if err != nil {
		return err
	}
	cur, err := getMin()
	if err == nil && strings.TrimSpace(cur) != "" {
		n, conv := strconv.Atoi(cur)
		if conv == nil && c < n {
			return fmt.Errorf("feed anti-rollback: feed %d < committed %d", c, n)
		}
	}
	return setMin(strconv.Itoa(c))
}

func Steps() []string { return []string{"feed-check", "index-sections", "export-occupancy"} }

func Enforce(title, body string, tags []string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("section title required")
	}
	section, feed, err := parse(body)
	if err != nil {
		return err
	}
	if section != "up" && section != "dn" {
		return fmt.Errorf("section must be up or dn")
	}
	if feed < 0 {
		return fmt.Errorf("feed must be >= 0")
	}
	if len(tags) == 0 {
		return fmt.Errorf("track-circuit tag required")
	}
	return nil
}

func ParseFeed(body string) (int, error) {
	_, c, err := parse(body)
	return c, err
}

func parse(body string) (section string, feed int, err error) {
	gotS, gotC := false, false
	for _, part := range strings.Fields(body) {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch k {
		case "section":
			section, gotS = v, true
		case "feed":
			n, conv := strconv.Atoi(v)
			if conv != nil {
				return "", 0, conv
			}
			feed, gotC = n, true
		}
	}
	if !gotS || !gotC {
		return "", 0, fmt.Errorf("need section= and feed=")
	}
	return section, feed, nil
}