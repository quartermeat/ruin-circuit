package main

import "testing"

func TestRespawnDelayScalesWithMatchAndDeaths(t *testing.T) {
	game := NewGame()
	if got := game.respawnDelay(0); got != 5*60 {
		t.Fatalf("initial delay is %d frames, want %d", got, 5*60)
	}
	if got := game.respawnDelay(1); got != 6*60 {
		t.Fatalf("first death delay is %d frames, want %d", got, 6*60)
	}
	game.matchFrames = 30 * 60
	if got := game.respawnDelay(2); got != 8*60 {
		t.Fatalf("match and death delay is %d frames, want %d", got, 8*60)
	}
	if got := game.respawnDelay(99); got != 16*60 {
		t.Fatalf("death delay cap is %d frames, want %d", got, 16*60)
	}
}

func TestNewGameStartsWithFreshDeathCounts(t *testing.T) {
	game := NewGame()
	if game.playerDeaths != 0 || game.aiHero.deaths != 0 {
		t.Fatalf("new game death counts were player=%d ai=%d", game.playerDeaths, game.aiHero.deaths)
	}
}
