package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

type OutcomeView struct {
	Name, Odd, Fair string
}

type FightView struct {
	N, Margin string
	Outcomes  []OutcomeView
}

type RowView struct {
	Name, Odd, Win, Price, Net, Class string
}

type PlanView struct {
	Title, Profit, Partial, Lose, Exp string
	Rows                              []RowView
}

type ViewData struct {
	HTML, Mode, K, M, Top, Bet string
	Error                      string
	EventName                  string
	Fights                     []FightView
	Plans                      []PlanView
	Done                       bool
}

func defaultView() ViewData {
	return ViewData{Mode: "best", K: "2", M: "3", Top: "10", Bet: "1"}
}

func process(v *ViewData) error {
	k, err1 := strconv.Atoi(v.K)
	m, err2 := strconv.Atoi(v.M)
	top, err3 := strconv.Atoi(v.Top)
	bet, err4 := strconv.ParseFloat(v.Bet, 64)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || k < 1 || m < 1 || bet <= 0 {
		return errors.New("k, m, top and bet must be valid numbers")
	}
	if k > 7 {
		return errors.New("k is limited to 7 on the web (more is too slow)")
	}
	if top < 1 || top > 100 {
		top = 10
	}

	src := strings.TrimSpace(v.HTML)
	if src == "" {
		return errors.New("paste the page HTML or the JSON")
	}

	in, err := ParseInput(src)
	if err != nil {
		return err
	}

	// bet from the JSON wins over the form field
	if in.Bet > 0 {
		bet = in.Bet
		v.Bet = strconv.FormatFloat(bet, 'f', -1, 64)
	}

	if k > len(in.Groups) {
		return fmt.Errorf("k=%d is bigger than the number of fights (%d)", k, len(in.Groups))
	}
	v.EventName = in.BetName
	v.Done = true

	for i, g := range in.Groups {
		n := Normalize(g)
		sum := 0.0
		fv := FightView{N: strconv.Itoa(i + 1)}
		for j, p := range g {
			sum += 1 / p.Odd
			fv.Outcomes = append(fv.Outcomes, OutcomeView{
				Name: p.Name,
				Odd:  fmt.Sprintf("%.3f", p.Odd),
				Fair: fmt.Sprintf("%.1f%%", n[j].Fair*100),
			})
		}
		fv.Margin = fmt.Sprintf("%.1f%%", (sum-1)*100)
		v.Fights = append(v.Fights, fv)
	}

	plans := Rank(in.Groups, bet, k, m, v.Mode)
	if top < len(plans) {
		plans = plans[:top]
	}
	for i, p := range plans {
		ids := make([]string, len(p.Fights))
		for j, f := range p.Fights {
			ids[j] = strconv.Itoa(f + 1)
		}
		pv := PlanView{
			Title:   fmt.Sprintf("#%d · fights %s · %d combos = $%.2f", i+1, strings.Join(ids, ", "), p.M, p.Stake),
			Profit:  fmt.Sprintf("%.1f%%", p.ProfitProb*100),
			Partial: fmt.Sprintf("%.1f%%", p.PartialProb*100),
			Lose:    fmt.Sprintf("%.1f%%", p.LoseAllProb*100),
			Exp:     fmt.Sprintf("%.1f%%", p.ExpReturn*100),
		}
		for _, b := range p.Bets {
			net := b.Price - p.Stake
			class := "win"
			if net <= 0 {
				class = "loss"
			}
			pv.Rows = append(pv.Rows, RowView{
				Name: b.Name(), Odd: fmt.Sprintf("%.2f", b.TotalOdd),
				Win: fmt.Sprintf("%.1f%%", b.WinProb*100), Price: fmt.Sprintf("$%.2f", b.Price),
				Net: fmt.Sprintf("%+.2f", net), Class: class,
			})
		}
		v.Plans = append(v.Plans, pv)
	}
	return nil
}

func render(c *gin.Context, status int, comp templ.Component) {
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = comp.Render(c.Request.Context(), c.Writer)
}

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		render(c, http.StatusOK, Page(defaultView()))
	})

	r.POST("/analyze", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
		v := ViewData{
			HTML: c.PostForm("html"),
			Mode: c.DefaultPostForm("mode", "best"),
			K:    c.DefaultPostForm("k", "2"),
			M:    c.DefaultPostForm("m", "3"),
			Top:  c.DefaultPostForm("top", "10"),
			Bet:  c.DefaultPostForm("bet", "1"),
		}
		if v.Mode != "best" && v.Mode != "big" && v.Mode != "chain" {
			v.Mode = "best"
		}
		status := http.StatusOK
		if err := process(&v); err != nil {
			v.Error = err.Error()
			status = http.StatusBadRequest
		}
		render(c, status, Page(v))
	})

	_ = r.Run(":8080")
}
