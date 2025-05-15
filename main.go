package main

import (
	"fmt"
	"math/rand"
)

// TODO: прогнать через GPT побольше нормальных имен
var (
	WarriorsNames = []string{"Артур", "Эльдрион", "Тарга"}
	MagesNames    = []string{"Мелисса", "Элла", "Кельси"}
	ArchersNames  = []string{"Алисия", "Кельси", "Мелисса"}
)

type TypeClass int
type TypeDebuff int

const (
	Warrior TypeClass = iota
	Mage
	Archer
)

const (
	None TypeDebuff = iota
	Skip
	Firing
)

type Player struct {
	Name          string
	IsAlive       bool
	Health        int
	MaxHealth     int
	Strength      int
	Type          TypeClass
	DebuffType    TypeDebuff
	OneGameDebuff bool
}

func (p *Player) Reset() {
	p.Health = p.MaxHealth
	p.DebuffType = None
	p.OneGameDebuff = false
}

func (p *Player) Attack(target *Player) {
	var random = rand.Intn(100)
	if random < 30 && !p.OneGameDebuff {
		p.Debuff(target)
	} else {
		fmt.Println(p.Name, " атакует ", target.Name, " и наносит ", p.Strength, " урона")
		target.Health -= p.Strength
	}
}

func (p *Player) Debuff(target *Player) {
	switch p.Type {
	case Warrior:
		fmt.Print(p.Name, " атакует ", target.Name, " и использует способность 'Удар возмездия'")
		fmt.Println("и наносит ", p.Strength+p.Strength*3/10, " урона")
		target.Health -= p.Strength + p.Strength*3/10
	case Archer:
		fmt.Print(p.Name, " атакует ", target.Name, " и использует способность 'Огненные стрелы' ")
		fmt.Println("и наносит ", p.Strength, " урона")
		target.Health -= p.Strength
		target.DebuffType = Firing
		p.OneGameDebuff = true
	case Mage:
		fmt.Print(p.Name, " атакует ", target.Name, " и использует способность 'Заворожение'")
		fmt.Println(", нанося ", p.Strength, " урона")
		target.Health -= p.Strength
		target.DebuffType = Skip
	}
}

func ProcessTurn(player1 *Player, player2 *Player) {
	if player1.DebuffType == Firing {
		player1.Health -= 2
		fmt.Println(player1.Name, " получает 2 урона от горения")
	}
	if player1.Health <= 0 {
		fmt.Println(player1.Name, " умирает")
		player1.IsAlive = false
		return
	}
	if player1.DebuffType == Skip {
		player1.DebuffType = None
		fmt.Println(player1.Name, " пропускает ход")
		return
	}
	player1.Attack(player2)

	if player2.DebuffType == Firing {
		player2.Health -= 2
		fmt.Println(player2.Name, " получает 2 урона от горения")
	}
	if player2.Health <= 0 {
		fmt.Println(player2.Name, " умирает")
		player2.IsAlive = false
		return
	}
	if player2.DebuffType == Skip {
		player2.DebuffType = None
		fmt.Println(player2.Name, " пропускает ход")
		return
	}
	player2.Attack(player1)

}

func GetTextType(t TypeClass) string {
	switch t {
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

func GetRandomPlayer() *Player {

	randomType := rand.Intn(3)
	randomIndex := rand.Intn(3)
	var playerType TypeClass
	var playerName string
	switch randomType {
	case 0:
		playerType = Warrior
		playerName = WarriorsNames[randomIndex]
	case 1:
		playerType = Archer
		playerName = ArchersNames[randomIndex]
	case 2:
		playerType = Mage
		playerName = MagesNames[randomIndex]
	}

	return &Player{
		Name:       playerName,
		Health:     50,
		MaxHealth:  rand.Intn(51) + 50,
		Strength:   rand.Intn(6) + 5,
		Type:       playerType,
		DebuffType: None,
		IsAlive:    true,
	}
}

func ResetPlayers(players []*Player) {
	for _, player := range players {
		player.Reset()
	}
}

func main() {
	fmt.Println("Добро пожаловать в Akvelon RPG SAGA!")
	var playersCount = 1

	for playersCount%2 != 0 {
		fmt.Println("Введите кол-во игроков")
		fmt.Scanln(&playersCount)

		if playersCount < 2 || playersCount%2 != 0 {
			fmt.Println("Кол-во игроков должно быть больше 1 и четным")
		}
	}

	players := make([]*Player, playersCount)

	for i := 0; i < playersCount; i++ {
		players[i] = GetRandomPlayer()
	}

	ResetPlayers(players)

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Список всех игроков:")
	for _, player := range players {
		fmt.Printf("Имя: %s, Тип: %s, Здоровье: %d, Сила: %d\n", player.Name, GetTextType(player.Type), player.Health, player.Strength)
	}
	fmt.Println("--------------------------------------------------------------------------------")

	for {
		rand.Shuffle(len(players), func(i, j int) {
			players[i], players[j] = players[j], players[i]
		})

		player1 := players[0]
		player2 := players[1]
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("Начинается битва между ", player1.Name, " ", GetTextType(player1.Type), " и ", player2.Name, " ", GetTextType(player2.Type))
		fmt.Println("--------------------------------------------------------------------------------")
		for player1.IsAlive && player2.IsAlive {
			ProcessTurn(player1, player2)
		}
		if player1.IsAlive {
			players = append(players[:1], players[2:]...)
		} else {
			players = append(players[:0], players[1:]...)
		}

		ResetPlayers(players)

		if len(players) == 1 {
			fmt.Println("--------------------------------------------------------------------------------")
			fmt.Println(players[0].Name, "выиграл")
			break
		}
	}
}
