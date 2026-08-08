package main

// respawnDelay is shared by both heroes so match pacing stays symmetric while
// each hero's own death count adds a small, capped penalty.
func (g *Game) respawnDelay(deaths int) int {
	matchSeconds := g.matchFrames / 60
	additionalSeconds := matchSeconds / 30
	if additionalSeconds > 25 {
		additionalSeconds = 25
	}
	deathSeconds := deaths
	if deathSeconds > 10 {
		deathSeconds = 10
	}
	return (5 + additionalSeconds + deathSeconds) * 60
}

func (g *Game) selectedAttackRange() float64 {
	if g.autoAttackRanged {
		return 190
	}
	return 58
}
