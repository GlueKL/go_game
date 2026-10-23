// Package rpg содержит игровую логику Akvelon RPG SAGA: игроков, бои и турнир.
// Логика не знает о графике: каждый шаг боя возвращает список событий,
// которые интерфейс затем анимирует.
package rpg

import (
	"math/rand"
	"strconv"
)

var (
	WarriorsNames = []string{"Артур", "Эльдрион", "Тарга", "Бьорн", "Гаррош", "Ульф"}
	MagesNames    = []string{"Мелисса", "Элла", "Кельси", "Морган", "Зарина", "Ирвин"}
	ArchersNames  = []string{"Алисия", "Леголин", "Сильва", "Робин", "Фрея", "Тауриэль"}
)

type Class int
type Debuff int

const (
	Warrior Class = iota
	Mage
	Archer
)

const (
	None Debuff = iota
	Skip
	Burning
)

// BurnDamage — урон от горения в начале каждого хода.
const BurnDamage = 2

// AbilityChance — шанс (в процентах) применить способность вместо обычной атаки.
const AbilityChance = 30

func (c Class) String() string {
	switch c {
	case Warrior:
		return "Воин"
	case Mage:
		return "Маг"
	case Archer:
		return "Лучник"
	default:
		return "Неизвестный класс"
	}
}

func (c Class) AbilityName() string {
	switch c {
	case Warrior:
		return "Удар возмездия"
	case Mage:
		return "Заворожение"
	case Archer:
		return "Огненные стрелы"
	default:
		return ""
	}
}

type Player struct {
	ID        int
	Name      string
	Class     Class
	Alive     bool
	Health    int
	MaxHealth int
	Strength  int
	Debuff    Debuff
	// AbilityUsed запрещает повторно применять способность до конца боя
	// (используется только лучником).
	AbilityUsed bool
	Wins        int
}

func (p *Player) Reset() {
	p.Health = p.MaxHealth
	p.Debuff = None
	p.AbilityUsed = false
}

// RandomPlayer создаёт бойца случайного класса. Имена из used не повторяются:
// когда свободных имён класса не осталось, к имени добавляется номер.
func RandomPlayer(rng *rand.Rand, id int, used map[string]bool) *Player {
	var (
		class Class
		names []string
	)
	switch rng.Intn(3) {
	case 0:
		class, names = Warrior, WarriorsNames
	case 1:
		class, names = Archer, ArchersNames
	default:
		class, names = Mage, MagesNames
	}
	var free []string
	for _, n := range names {
		if !used[n] {
			free = append(free, n)
		}
	}
	name := names[rng.Intn(len(names))]
	if len(free) > 0 {
		name = free[rng.Intn(len(free))]
	}
	for i := 2; used[name]; i++ {
		name = names[rng.Intn(len(names))] + " " + strconv.Itoa(i)
	}
	if used != nil {
		used[name] = true
	}
	p := &Player{
		ID:        id,
		Name:      name,
		Class:     class,
		Alive:     true,
		MaxHealth: rng.Intn(51) + 50,
		Strength:  rng.Intn(6) + 5,
	}
	p.Reset()
	return p
}

type EventKind int

const (
	EvAttack  EventKind = iota // обычная атака
	EvAbility                  // атака со способностью
	EvBurn                     // урон от горения
	EvSkip                     // пропуск хода из-за заворожения
	EvDeath                    // смерть
)

// Event описывает одно действие в бою. Subject — игрок, у которого изменилось
// состояние: цель атаки, либо сам действующий для горения, пропуска и смерти.
// HealthAfter и DebuffAfter — состояние Subject сразу после события.
type Event struct {
	Kind        EventKind
	Actor       *Player
	Subject     *Player
	Damage      int
	HealthAfter int
	DebuffAfter Debuff
	Text        string
}

type Battle struct {
	Fighters [2]*Player
	Winner   *Player
	Loser    *Player
	turn     int
}

func NewBattle(a, b *Player) *Battle {
	return &Battle{Fighters: [2]*Player{a, b}}
}

func (b *Battle) Over() bool { return b.Winner != nil }

// Step выполняет ход очередного бойца и возвращает произошедшие события.
func (b *Battle) Step(rng *rand.Rand) []Event {
	if b.Over() {
		return nil
	}
	actor := b.Fighters[b.turn%2]
	target := b.Fighters[(b.turn+1)%2]
	b.turn++

	var events []Event
	if actor.Debuff == Burning {
		actor.Health -= BurnDamage
		events = append(events, Event{
			Kind: EvBurn, Actor: actor, Subject: actor, Damage: BurnDamage,
			HealthAfter: actor.Health, DebuffAfter: actor.Debuff,
			Text: actor.Name + " получает 2 урона от горения",
		})
		if actor.Health <= 0 {
			return append(events, b.kill(actor, target))
		}
	}
	if actor.Debuff == Skip {
		actor.Debuff = None
		return append(events, Event{
			Kind: EvSkip, Actor: actor, Subject: actor,
			HealthAfter: actor.Health, DebuffAfter: actor.Debuff,
			Text: actor.Name + " заворожен и пропускает ход",
		})
	}

	events = append(events, attack(rng, actor, target))
	if target.Health <= 0 {
		events = append(events, b.kill(target, actor))
	}
	return events
}

func (b *Battle) kill(dead, winner *Player) Event {
	dead.Alive = false
	b.Winner, b.Loser = winner, dead
	winner.Wins++
	return Event{
		Kind: EvDeath, Actor: dead, Subject: dead,
		HealthAfter: dead.Health, DebuffAfter: dead.Debuff,
		Text: dead.Name + " погибает",
	}
}

func attack(rng *rand.Rand, p, target *Player) Event {
	ev := Event{Kind: EvAttack, Actor: p, Subject: target, Damage: p.Strength}
	if rng.Intn(100) < AbilityChance && !p.AbilityUsed {
		ev.Kind = EvAbility
		switch p.Class {
		case Warrior:
			ev.Damage = p.Strength + p.Strength*3/10
		case Archer:
			target.Debuff = Burning
			p.AbilityUsed = true
		case Mage:
			target.Debuff = Skip
		}
	}
	target.Health -= ev.Damage
	ev.HealthAfter = target.Health
	ev.DebuffAfter = target.Debuff
	if ev.Kind == EvAbility {
		ev.Text = p.Name + " применяет «" + p.Class.AbilityName() + "» и наносит " + strconv.Itoa(ev.Damage) + " урона"
	} else {
		ev.Text = p.Name + " атакует " + target.Name + " и наносит " + strconv.Itoa(ev.Damage) + " урона"
	}
	return ev
}

// Tournament — турнир на выбывание: случайные пары дерутся, пока не останется один.
type Tournament struct {
	Players  []*Player // все участники в исходном порядке
	Alive    []*Player
	Battle   *Battle
	Number   int // номер текущего боя, начиная с 1
	Champion *Player
	rng      *rand.Rand
}

func NewTournament(rng *rand.Rand, count int) *Tournament {
	t := &Tournament{rng: rng}
	used := map[string]bool{}
	for i := 0; i < count; i++ {
		p := RandomPlayer(rng, i, used)
		t.Players = append(t.Players, p)
		t.Alive = append(t.Alive, p)
	}
	return t
}

func (t *Tournament) Rng() *rand.Rand { return t.rng }

// NextBattle выбирает двух случайных живых бойцов и начинает бой.
func (t *Tournament) NextBattle() *Battle {
	if len(t.Alive) < 2 {
		return nil
	}
	t.rng.Shuffle(len(t.Alive), func(i, j int) {
		t.Alive[i], t.Alive[j] = t.Alive[j], t.Alive[i]
	})
	t.Number++
	t.Battle = NewBattle(t.Alive[0], t.Alive[1])
	return t.Battle
}

// FinishBattle убирает проигравшего и восстанавливает выживших.
func (t *Tournament) FinishBattle() {
	if t.Battle == nil || !t.Battle.Over() {
		return
	}
	for i, p := range t.Alive {
		if p == t.Battle.Loser {
			t.Alive = append(t.Alive[:i], t.Alive[i+1:]...)
			break
		}
	}
	for _, p := range t.Alive {
		p.Reset()
	}
	t.Battle = nil
	if len(t.Alive) == 1 {
		t.Champion = t.Alive[0]
	}
}
