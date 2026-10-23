package rpg

import (
	"math/rand"
	"testing"
)

func TestTournamentProducesChampion(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		tr := NewTournament(rng, 8)
		for battles := 0; tr.Champion == nil; battles++ {
			if battles > 7 {
				t.Fatalf("seed %d: too many battles", seed)
			}
			b := tr.NextBattle()
			for steps := 0; !b.Over(); steps++ {
				if steps > 1000 {
					t.Fatalf("seed %d: battle does not end", seed)
				}
				for _, ev := range b.Step(rng) {
					if ev.HealthAfter != ev.Subject.Health && ev.Kind != EvDeath {
						t.Fatalf("seed %d: event health %d != actual %d", seed, ev.HealthAfter, ev.Subject.Health)
					}
				}
			}
			if b.Loser.Alive || !b.Winner.Alive || b.Loser.Health > 0 {
				t.Fatalf("seed %d: bad battle result", seed)
			}
			tr.FinishBattle()
		}
		wins := 0
		for _, p := range tr.Players {
			wins += p.Wins
		}
		if len(tr.Alive) != 1 || wins != 7 {
			t.Fatalf("seed %d: alive %d, total wins %d", seed, len(tr.Alive), wins)
		}
	}
}

func TestArcherAbilityOncePerBattle(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	archer := &Player{Name: "A", Class: Archer, Alive: true, MaxHealth: 1000, Strength: 1}
	dummy := &Player{Name: "D", Class: Warrior, Alive: true, MaxHealth: 1000, Strength: 0}
	archer.Reset()
	dummy.Reset()
	b := NewBattle(archer, dummy)
	abilities := 0
	for i := 0; i < 200; i++ {
		for _, ev := range b.Step(rng) {
			if ev.Kind == EvAbility && ev.Actor == archer {
				abilities++
				if ev.DebuffAfter != Burning {
					t.Fatal("archer ability must set burning")
				}
			}
		}
	}
	if abilities != 1 {
		t.Fatalf("archer used ability %d times, want 1", abilities)
	}
}

func TestUniqueNames(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	tr := NewTournament(rng, 40)
	seen := map[string]bool{}
	for _, p := range tr.Players {
		if seen[p.Name] {
			t.Fatalf("duplicate name %q", p.Name)
		}
		seen[p.Name] = true
	}
}
