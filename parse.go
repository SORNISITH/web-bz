package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"encoding/json"
	"golang.org/x/net/html"
)

// ---------- layouts: one entry per HTML structure ----------

// Team: 0 = first name, 1 = second name, -1 = fixed Label (for example "Draw")
type Outcome struct {
	Key   string
	Team  int
	Label string
}

type Layout struct {
	Sport       string
	Card        string
	Link        string
	Name        string
	Market      string
	MarketName  string
	MarketValue string
	Locked      string // optional: class of a suspended odds box, which is skipped
	Outcomes    []Outcome
}



var layouts = []Layout{
	{
		Sport: "ufc",
		Card:  "ufc-fight-card", Link: "ufc-fight-card__link",
		Name:   "ufc-fighter__name",
		Market: "ufc-fight-markets__market", MarketName: "ufc-fight-markets__name",
		MarketValue: "ui-market__value",
		Outcomes: []Outcome{ // X on this layout is not a win odd, so it is skipped
			{Key: "W1", Team: 0},
			{Key: "W2", Team: 1},
		},
	},
	{
		Sport: "football",
		Card:  "top-events-game-card", Link: "top-events-game-card__link",
		Name:   "top-events-game-card-scoreboard-team__caption",
		Market: "top-events-game-card-markets__item", MarketName: "ui-market__name",
		MarketValue: "ui-market__value",
		Outcomes: []Outcome{
			{Key: "W1", Team: 0},
			{Key: "X", Team: -1, Label: "Draw"},
			{Key: "W2", Team: 1},
		},
	},
	// add a new sport here: copy an entry and change the class names
}

// ---------- small HTML helpers ----------

func hasClass(n *html.Node, class string) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(text(c))
	}
	return strings.TrimSpace(sb.String())
}

func findAll(n *html.Node, class string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if hasClass(x, class) {
			out = append(out, x)
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func findFirst(n *html.Node, class string) *html.Node {
	if r := findAll(n, class); len(r) > 0 {
		return r[0]
	}
	return nil
}

var eventRe = regexp.MustCompile(`/line/[^/]+/\d+-([^/]+)/`)

// "/en/line/ufc/3072381-ufc-332/..." -> "ufc 332"
// "/en/top-events/la-liga/line/football/127733-spain-la-liga/..." -> "spain la liga"
func eventName(href string) string {
	m := eventRe.FindStringSubmatch(href)
	if m == nil {
		return ""
	}
	return strings.ReplaceAll(m[1], "-", " ")
}

// ---------- parsing ----------


func parseCard(card *html.Node, l Layout) []Player {
	var names []string

	for _, n := range findAll(card, l.Name) {
		names = append(names, text(n))
	}

	if len(names) != 2 {
		return nil
	}

	odds := map[string]float64{}

	// Parse markets
	for _, m := range findAll(card, l.Market) {

		// Skip suspended / locked market
		if l.Locked != "" && hasClass(m, l.Locked) {
			continue
		}

		nameNode := findFirst(m, l.MarketName)
		valNode := findFirst(m, l.MarketValue)

		if nameNode == nil || valNode == nil {
			continue
		}

		v, err := strconv.ParseFloat(text(valNode), 64)
		if err != nil || v <= 1 {
			continue
		}

		odds[text(nameNode)] = v
	}

	var group []Player

	for _, o := range l.Outcomes {
		v, ok := odds[o.Key]
		if !ok {
			continue
		}

		name := o.Label

		if o.Team >= 0 {
			name = names[o.Team]
		}

		group = append(group, Player{
			Name: name,
			Odd:  v,
		})
	}

	// Need both sides.
	// A draw alone is not a match.
	if len(group) < 2 ||
		odds["W1"] == 0 ||
		odds["W2"] == 0 {
		return nil
	}

	return group
}
// ParseFights tries every layout and joins what it finds
func ParseFights(src string) (Input, error) {
	out := Input{BetName: ""}

	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return out, fmt.Errorf("bad html: %w", err)
	}

	var sports []string
	for _, l := range layouts {
		cards := findAll(doc, l.Card)
		if len(cards) == 0 {
			continue
		}
		found := 0
		for _, card := range cards {
			if out.BetName == "" {
				if link := findFirst(card, l.Link); link != nil {
					out.BetName = eventName(attr(link, "href"))
				}
			}
			if g := parseCard(card, l); g != nil {
				out.Groups = append(out.Groups, g)
				found++
			}
		}
		if found > 0 {
			sports = append(sports, l.Sport)
		}
	}

	if len(out.Groups) == 0 {
		return out, fmt.Errorf("no matches with odds found (unknown layout, or the page is rendered by JavaScript: paste the HTML from DevTools)")
	}
	if out.BetName == "" {
		out.BetName = "unknown"
	}
	out.BetName += " (" + strings.Join(sports, ", ") + ")"
	return out, nil
}



func ParseJSON(src string) (Input, error) {
	var in Input
	if err := json.Unmarshal([]byte(src), &in); err != nil {
		return in, fmt.Errorf("bad JSON: %w", err)
	}
	if len(in.Groups) == 0 {
		return in, fmt.Errorf("JSON has no groups")
	}
	for i, g := range in.Groups {
		if len(g) < 2 {
			return in, fmt.Errorf("group %d needs at least 2 players", i+1)
		}
		for _, p := range g {
			if p.Name == "" {
				return in, fmt.Errorf("group %d has a player without a name", i+1)
			}
			if p.Odd <= 1 {
				return in, fmt.Errorf("%s in group %d: odd must be greater than 1 (got %v)", p.Name, i+1, p.Odd)
			}
		}
	}
	if in.BetName == "" {
		in.BetName = "json"
	}
	return in, nil
}

// JSON if it starts with "{", otherwise page HTML
func ParseInput(src string) (Input, error) {
	if strings.HasPrefix(strings.TrimSpace(src), "{") {
		return ParseJSON(src)
	}
	return ParseFights(src)
}
