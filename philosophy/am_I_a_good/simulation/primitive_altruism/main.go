package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sort"
)

const (
	requiredEnergy = 1000
	amountOfPortions = 100
	portionEnergy = 2000
	tries = 1000
	familySize = 100
	extraFoodEfficiency = 1
)

type strategyShareEverything struct{}

func (strategy *strategyShareEverything) HandleFood(
	player *Player,
	food *Food,
) []Action {
	var actions []Action

	oneShare := food.Amount / uint(len(player.Family.Players))
	for _, relative := range player.Family.Players {
		actions = append(actions, Action{
			ActionType:  ActionTypeEat,
			Amount:      oneShare,
			Destination: relative,
		})
	}

	rest := food.Amount - oneShare * uint(len(player.Family.Players))
	if rest > 0 {
		actions = append(actions, Action{
			ActionType:  ActionTypeEat,
			Amount:      rest,
			Destination: player,
		})
	}

	return actions
}

type strategyEatTheRest struct{}

func (strategy *strategyEatTheRest) HandleFood(
	player *Player,
	food *Food,
) []Action {
	return []Action{{ActionTypeEat, food.Amount, player}}
}

type strategyHideTheRest struct{}

func (strategy *strategyHideTheRest) HandleFood(
	player *Player,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - player.HasEnergy
	if toSurvive > 0 {
		eatAmount := toSurvive
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, player})
		amount -= eatAmount
	}
	for amount > 0 {
		toHide := amount
		if toHide > 100 {
			toHide = 100
		}
		result = append(result, Action{ActionTypeHide, toHide, player})
		amount -= toHide
	}
	return result
}

type strategyShareAndHideTheRest struct{}

func (strategy *strategyShareAndHideTheRest) HandleFood(
	player *Player,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - player.HasEnergy
	if toSurvive > 0 {
		eatAmount := toSurvive
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, player})
		amount -= eatAmount
	}

	// save those who we can save

	var candidates []*Player
	for _, relative := range player.Family.Players {
		hasEnergy := relative.TotalEnergy()
		if hasEnergy < requiredEnergy {
			candidates = append(candidates, relative)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		totalEnergyI := candidates[i].TotalEnergy()
		totalEnergyJ := candidates[j].TotalEnergy()
		return totalEnergyI > totalEnergyJ
	})
	for _, candidate := range candidates {
		hasEnergy := candidate.TotalEnergy()
		toSurvive := requiredEnergy - hasEnergy
		if toSurvive > amount {
			toSurvive = amount
		}
		result = append(result, Action{ActionTypeEat, toSurvive, candidate})
		amount -= toSurvive
		if amount == 0 {
			break
		}
	}

	// hide the rest

	if amount > 0 {
		result = append(result, Action{ActionTypeHide, amount, player})
	}
	return result
}

type strategyShareTheRest struct{}

func (strategy *strategyShareTheRest) HandleFood(
	player *Player,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - player.HasEnergy
	if toSurvive > 0 {
		eatAmount := toSurvive
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, player})
		amount -= eatAmount
	}

	// save those who we can save

	var candidates []*Player
	for _, relative := range player.Family.Players {
		hasEnergy := relative.TotalEnergy()
		if hasEnergy < requiredEnergy {
			candidates = append(candidates, relative)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		totalEnergyI := candidates[i].TotalEnergy()
		totalEnergyJ := candidates[j].TotalEnergy()
		return totalEnergyI > totalEnergyJ
	})
	for _, candidate := range candidates {
		hasEnergy := candidate.TotalEnergy()
		toSurvive := requiredEnergy - hasEnergy
		if toSurvive > amount {
			toSurvive = amount
		}
		result = append(result, Action{ActionTypeEat, toSurvive, candidate})
		amount -= toSurvive
		if amount == 0 {
			break
		}
	}

	if amount == 0 {
		return result
	}

	// use the rest food to equalize the eat energy

	// calculating the minimal amount of food everybody should get, part 1
	var energies []uint
	for _, candidate := range player.Family.Players {
		if candidate.TotalEnergy() < requiredEnergy {
			// they will be dead anyway
			continue
		}
		hadEat := candidate.HadEat
		for _, action := range result {
			if action.Destination == candidate {
				hadEat += action.Amount
			}
		}
		energies = append(energies, hadEat)
	}
	sort.Slice(energies, func(i, j int) bool {
		return energies[i] < energies[j]
	})

	if len(energies) > 0 {
		// calculating the minimal amount of food everybody should get, part 2
		feedeesCount := uint(0)
		lowEatBar := uint(0)
		energies = append(energies, energies[len(energies)-1]+food.Amount) // adding a fake element to the end
		prevEnergy := energies[0]
		for _, energy := range energies[1:] {
			feedeesCount++

			if energy == prevEnergy {
				continue
			}

			energyDiff := energy - prevEnergy
			if energyDiff * feedeesCount > amount {
				toEat := amount / feedeesCount
				lowEatBar = prevEnergy + toEat
				amount -= toEat * feedeesCount
				break
			}
			lowEatBar = energy
			amount -= feedeesCount * energyDiff
			prevEnergy = energy
		}
		if amount > feedeesCount {
			toEat := amount / feedeesCount
			lowEatBar += toEat * feedeesCount
			amount -= toEat * feedeesCount
		}

		// sharing the food
		for _, candidate := range player.Family.Players {
			if candidate.TotalEnergy() < requiredEnergy {
				// they will be dead anyway
				continue
			}

			hadEat := candidate.HadEat
			for _, action := range result {
				if action.Destination == candidate {
					hadEat += action.Amount
				}
			}

			if hadEat >= lowEatBar {
				continue
			}

			shareAmount := lowEatBar - hadEat

			result = append(result, Action{ActionTypeEat, shareAmount, candidate})
		}
	}

	// if something left, the eat the rest
	if amount <= 0 {
		return result
	}

	result = append(result, Action{ActionTypeEat, amount, player})
	return result
}

type ActionType uint

const (
	ActionTypeUndefined = ActionType(iota)
	ActionTypeEat
	ActionTypeHide
)

type Food struct {
	AlreadyHidden bool
	Amount uint
}

type Action struct {
	ActionType
	Amount      uint
	Destination *Player
}

type Strategy interface {
	HandleFood(player *Player, food *Food) []Action
}

type Player struct {
	Family    *Family
	HadEat    uint
	HasEnergy uint
	OwnsFood  uint
}

func (player *Player) HandleFood(food *Food) []Action {
	return player.Family.Strategy.HandleFood(player, food)
}

func (player *Player) EatEnergy() uint {
	hadEat := player.HadEat
	if hadEat <= 1000 {
		return hadEat
	}
	return uint(float64(requiredEnergy) + float64(hadEat - requiredEnergy) * extraFoodEfficiency)
}

func (player *Player) TotalEnergy() uint {
	return player.HasEnergy + player.OwnsFood + player.EatEnergy()
}

type Family struct {
	Playground *Playground
	Players    []*Player
	Strategy   Strategy
}

func (family *Family) AddPlayer() {
	family.Players = append(family.Players, &Player{Family: family})
}

func (family *Family) RemovePlayer(removePlayer *Player) {
	for playerIdx, player := range family.Players {
		if player != removePlayer {
			continue
		}

		family.Players[playerIdx] = family.Players[len(family.Players)-1]
		family.Players = family.Players[:len(family.Players)-1]
	}
}

type Playground struct {
	Families []*Family
}

func (playground *Playground) AddFamily(strategy Strategy, playerAmount uint) {
	family := &Family{Playground: playground, Strategy: strategy}
	for i := uint(0); i < playerAmount; i++ {
		family.AddPlayer()
	}
	playground.Families = append(playground.Families, family)
}

func (playground *Playground) Players() []*Player {
	var result []*Player
	for _, family := range playground.Families {
		result = append(result, family.Players...)
	}
	return result
}

func (playground *Playground) IterateWeek() {
	players := playground.Players()

	var foundFood []*Food
	for i := 0; i < amountOfPortions; i++ {
		foundFood = append(foundFood, &Food{false, portionEnergy})
	}

	newPlayerFood := make([][]*Food, len(players))
	for playerIdx, player := range players {
		if player.OwnsFood == 0 {
			continue
		}
		newPlayerFood[playerIdx] = append(newPlayerFood[playerIdx], &Food{true, player.OwnsFood})
		player.OwnsFood = 0
	}

	for len(foundFood) > 0 {
		for _, foodPortion := range foundFood {
			playerIdx := rand.Intn(len(players))
			newPlayerFood[playerIdx] = append(newPlayerFood[playerIdx], foodPortion)
		}
		foundFood = foundFood[:0]

		for playerIdx, player := range players {
			newFood := newPlayerFood[playerIdx]
			newPlayerFood[playerIdx] = newPlayerFood[playerIdx][:0]

			for foodIdx := range newFood {
				foodPortion := newFood[len(newFood) - foodIdx-1] // first we handle non-hidden food

				actions := player.HandleFood(foodPortion)
				usedFood := uint(0)
				for _, action := range actions {
					if action.Amount > foodPortion.Amount-usedFood {
						panic(fmt.Sprintf("cheater! %+v: %d > %+v - %d (%T)",
							action, action.Amount, foodPortion, usedFood, player.Family.Strategy))
					}
					usedFood += action.Amount
					switch action.ActionType {
					case ActionTypeEat:
						action.Destination.HadEat += action.Amount
					case ActionTypeHide:
						if rand.Intn(2) == 0 && !foodPortion.AlreadyHidden {
							foundFood = append(foundFood, &Food{false, action.Amount})
						} else {
							action.Destination.OwnsFood += action.Amount
						}
					default:
						panic("unknown action")
					}
				}
				if usedFood != foodPortion.Amount {
					panic(fmt.Sprintf("something is wrong: %d != %+v (%T)",
						usedFood, foodPortion, player.Family.Strategy))
				}
			}
		}
	}

	for _, player := range players {
		player.HasEnergy += player.EatEnergy()
		player.HadEat = 0
		if player.HasEnergy < requiredEnergy {
			if player.TotalEnergy() >= requiredEnergy {
				panic(fmt.Sprintf("invalid strategy: %+v", player))
			}
			player.Family.RemovePlayer(player)
			continue
		}
		player.HasEnergy -= requiredEnergy
	}
}

func main() {
	var totalPopulation [3]uint64

	go func() {
		log.Fatal(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	for i := 0; i < tries; i++ {
		playground := &Playground{}
		playground.AddFamily(&strategyEatTheRest{}, familySize)
		playground.AddFamily(&strategyShareAndHideTheRest{}, familySize)
		playground.AddFamily(&strategyShareEverything{}, familySize)

		for week := 0; week < 55; /* one year */ week++ {
			playground.IterateWeek()
		}

		for familyIdx, family := range playground.Families {
			totalPopulation[familyIdx] += uint64(len(family.Players))
		}
	}

	for strategyIdx := 0; strategyIdx<len(totalPopulation); strategyIdx++ {
		survived := totalPopulation[strategyIdx]
		fmt.Printf("strategy #%d: sum of survived in %d tries: %d. Survival rate: %.0f%%\n",
			strategyIdx+1, tries, survived, float64(survived)/float64(tries)/familySize*100)
	}
}
