package sim

// Result is a combat result.
type Result uint8

// Combat results.
const (
	NoEffect Result = iota
	AttackerLoses
	Exchange
	DefenderRetreats
	DefenderLoses
)

// String is the result's code, as a report shows it.
func (r Result) String() string {
	return [...]string{"NE", "AL", "EX", "DR", "DL"}[r]
}

// Columns of the table: the odds, attacker to defender.
const (
	col1to2 = iota // worse than 1:1
	col1to1
	col2to1
	col3to1
	col4to1
	columns
)

// crtTable is the one combat-results table (docs/PLAN.md §6.3.1), by d6 roll and column.
// Better odds or a higher roll never gives the attacker a worse result.
var crtTable = [6][columns]Result{
	// 1:2           1:1               2:1               3:1               4:1
	{AttackerLoses, AttackerLoses, NoEffect, Exchange, DefenderRetreats},         // 1
	{AttackerLoses, NoEffect, Exchange, DefenderRetreats, DefenderLoses},         // 2
	{AttackerLoses, NoEffect, DefenderRetreats, DefenderRetreats, DefenderLoses}, // 3
	{NoEffect, Exchange, DefenderRetreats, DefenderLoses, DefenderLoses},         // 4
	{NoEffect, DefenderRetreats, DefenderLoses, DefenderLoses, DefenderLoses},    // 5
	{Exchange, DefenderRetreats, DefenderLoses, DefenderLoses, DefenderLoses},    // 6
}

// Column is the odds column for an attack, shifted in the defender's favour by terrain.
func Column(attack, defence int, t Terrain) int {
	col := col1to2
	switch {
	case attack >= 4*defence:
		col = col4to1
	case attack >= 3*defence:
		col = col3to1
	case attack >= 2*defence:
		col = col2to1
	case attack >= defence:
		col = col1to1
	}
	return max(col-int(t), col1to2)
}

// CRT resolves one attack with a d6 roll (1 to 6).
func CRT(attack, defence int, t Terrain, roll int) Result {
	return crtTable[min(max(roll, 1), 6)-1][Column(attack, defence, t)]
}
