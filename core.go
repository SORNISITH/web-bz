package main

import (
	"cmp"
	"math"
	"slices"
	"strings"
)

type Player struct {
	Name string  `json:"name"`
	Odd  float64 `json:"odd"`
	Fair float64 `json:"-"`
}


type Input struct {
	BetName string     `json:"bet_name"`
	Bet     float64    `json:"bet"`
	Groups  [][]Player `json:"groups"`
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

func Normalize(group []Player) []Player {
	sum := 0.0
	for _, p := range group {
		sum += 1 / p.Odd
	}
	out := make([]Player, len(group))
	for i, p := range group {
		p.Fair = (1 / p.Odd) / sum
		out[i] = p
	}
	return out
}

type Versus struct {
	Opponents []Player
	TotalOdd  float64
	WinProb   float64
	Price     float64
}

type MatchList []Versus

func NewVersus(players []Player, bet float64) Versus {
	odd, prob := 1.0, 1.0
	for _, p := range players {
		odd *= p.Odd
		prob *= p.Fair
	}
	odd = round2(odd)
	return Versus{Opponents: players, TotalOdd: odd, WinProb: prob, Price: round2(bet * odd)}
}

func (v Versus) Name() string {
	n := make([]string, len(v.Opponents))
	for i, p := range v.Opponents {
		n[i] = p.Name
	}
	return strings.Join(n, " : ")
}

func (h MatchList) SortByPrice() {
	slices.SortStableFunc(h, func(a, b Versus) int { return cmp.Compare(b.Price, a.Price) })
}

// one player from EACH group
func Stacks(bet float64, groups ...[]Player) MatchList {
	if len(groups) == 0 {
		return nil
	}
	combos := [][]Player{{}}
	for _, g := range groups {
		g = Normalize(g)
		var next [][]Player
		for _, c := range combos {
			for _, p := range g {
				n := make([]Player, len(c), len(c)+1)
				copy(n, c)
				next = append(next, append(n, p))
			}
		}
		combos = next
	}
	out := make(MatchList, 0, len(combos))
	for _, c := range combos {
		out = append(out, NewVersus(c, bet))
	}
	return out
}

func combinations(n, k int) [][]int {
	var out [][]int
	var cur []int
	var rec func(start int)
	rec = func(start int) {
		if len(cur) == k {
			out = append(out, slices.Clone(cur))
			return
		}
		for i := start; i < n; i++ {
			cur = append(cur, i)
			rec(i + 1)
			cur = cur[:len(cur)-1]
		}
	}
	rec(0)
	return out
}

type Plan struct {
	Fights      []int // 0-based
	M           int
	Stake       float64
	Bets        MatchList
	ProfitProb  float64
	PartialProb float64
	LoseAllProb float64
	ExpReturn   float64
}

func EvalPlan(groups [][]Player, bet float64, idx []int, m int) Plan {
	sub := make([][]Player, len(idx))
	for i, gi := range idx {
		sub[i] = groups[gi]
	}
	all := Stacks(bet, sub...)
	all.SortByPrice()

	m = min(m, len(all))
	p := Plan{Fights: idx, M: m, Stake: bet * float64(m), Bets: slices.Clone(all[:m])}

	exp := 0.0
	for i, v := range all {
		if i >= m {
			p.LoseAllProb += v.WinProb
			continue
		}
		exp += v.WinProb * v.Price
		if v.Price > p.Stake {
			p.ProfitProb += v.WinProb
		} else {
			p.PartialProb += v.WinProb
		}
	}
	p.ExpReturn = exp / p.Stake
	return p
}

// mode: "best" (lowest risk), "big" (biggest win), "chain" (best weakest win)
func Rank(groups [][]Player, bet float64, k, m int, mode string) []Plan {
	plans := make([]Plan, 0)
	for _, idx := range combinations(len(groups), k) {
		plans = append(plans, EvalPlan(groups, bet, idx, m))
	}
	last := func(p Plan) float64 { return p.Bets[len(p.Bets)-1].Price / p.Stake }

	slices.SortStableFunc(plans, func(a, b Plan) int {
		switch mode {
		case "big":
			return cmp.Or(
				cmp.Compare(b.Bets[0].Price-b.Stake, a.Bets[0].Price-a.Stake),
				cmp.Compare(b.ProfitProb, a.ProfitProb),
			)
		case "chain":
			return cmp.Or(
				cmp.Compare(last(b), last(a)),
				cmp.Compare(b.ProfitProb, a.ProfitProb),
			)
		default:
			return cmp.Or(
				cmp.Compare(b.ProfitProb, a.ProfitProb),
				cmp.Compare(a.LoseAllProb, b.LoseAllProb),
				cmp.Compare(b.ExpReturn, a.ExpReturn),
			)
		}
	})
	return plans
}
