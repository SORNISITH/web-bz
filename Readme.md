# Odds Chain Analyzer

A small web tool that takes the odds for a set of matches or fights, builds
chains (combos) of outcomes, and shows the chance, cost and expected return
of each plan, so you can compare bets before placing them.

## Quick start

1. Open the page.
2. Add your events, either by **pasting JSON** (Paste tab) or by typing them in (Build tab).
3. Pick a mode and set the numbers.
4. Click **Analyze**.

Your last input and settings are saved in your browser and restored on your next visit.

## Entering data
### Option A: Build tab (no JSON needed)
1. Click **Build**.
2. Type a bet name and a stake.
3. Each **group** is one match or fight. Add the team names and their odds.
4. Use **+ Add team** for more outcomes in a group, **+ Add group** for another match,
   and **✕** / **Remove group** to delete.
5. Rows with an empty name or odd are skipped automatically.

The Build tab writes JSON into the Paste tab as you type. Switch back to
**Paste** at any time to see or copy it.

### Option B: Paste tab

Paste JSON in this format:

```json
{
  "bet_name": "football ",
  "bet": 1,
  "groups": [
    [
      { "name": "Netherlands", "odd": 2.083 },
      { "name": "Greece", "odd": 3.505 }
    ],
    [
      { "name": "Portugal", "odd": 2.022 },
      { "name": "Denmark", "odd": 3.7 }
    ]
  ]
}
```

| Field      | Meaning                                              |
|------------|------------------------------------------------------|
| `bet_name` | Title shown above the results                        |
| `bet`      | Stake                                                |
| `groups`   | One list per match; each item is an outcome          |
| `name`     | Team or fighter name                                 |
| `odd`      | Decimal odds (for example 2.083)                     |

You can also paste a page's HTML, and the tool will try to read it.
(The Build tab only understands JSON.)

## Settings

| Setting             | What it does                                               |
|---------------------|------------------------------------------------------------|
| **Mode: best**      | Low risk: favors plans that lose less often                |
| **Mode: big**       | Big win: favors the largest possible payout                |
| **Mode: chain**     | Best weakest win: improves the worst winning case          |
| **Fights per chain (k)** | How many matches are joined in one combo (1-7)        |
| **Combos to bet (m)**    | How many combos are included in one plan              |
| **Show top**        | How many plans to display (1-100)                          |
| **Bet $ per combo** | Money placed on each combo                                 |

## Reading the results

**Events table:** every match with its outcomes, their odds, the "fair" odds
(bookmaker margin removed) and the bookmaker's margin.

**Each plan shows:**

- **Profit:** chance that the plan ends in profit
- **Win but loss:** chance that some combos win but you still lose money overall
- **Lose all:** chance that every combo loses
- **Expected:** average return over many repeats

**Combo table:** the combo, its combined odd, its chance, its price, and the
net result if it wins (green = profit, red = loss).

## Tips

- Start with `k = 2` or `3`. Long chains win rarely.
- Compare the same data in all three modes.
- Keep `m` small so the stake stays under control.
- Check spelling in the Build tab; names are saved exactly as typed.

## Important
Chances are **estimates** based on odds with the margin removed. The expected
return stays below 100%, so **no plan is a guaranteed win**. Only bet what you
can afford to lose.
