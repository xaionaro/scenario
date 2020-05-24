package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"sync"

	"github.com/xaionaro-go/rand/mathrand"
)

const (
	requiredEnergy = 1000
	amountOfPortions = 500/3
	portionEnergy = 1500
	tries = 100
	familySize = 100
	extraFoodEfficiency = 1
	personGraduationInWeeks = 16 * 54
	personExpirationInWeeks = 80 * 54
	startBabyEnergy = 50000
	createBabyEnergy = 40000
	enableChildren = true
	enableAging = true
)

type randSourceT struct {
	mathrand.PRNG
}
var prng = randSourceT{*mathrand.New()}
func (randSource *randSourceT) Seed(seed int64) {
	randSource.SetSeed(uint64(seed))
}
func (randSource *randSourceT) Int63() int64 {
	return int64(randSource.Uint64AddRotateMultiply() >> 1)
	//return int64(randSource.Uint64Xoshiro256() >> 1)
}
var shuffler = rand.New(&prng)

func init() {
	rand.Seed(0)
}

type strategyDoNotTrust struct{}

func (strategy *strategyDoNotTrust) HandleFood(
	citizen *Citizen,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - int64(citizen.HasEnergy)
	if toSurvive > 0 {
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &citizen.Person, "self-preservation"})
		amount -= eatAmount
	}
	sort.Slice(citizen.Children, func(i, j int) bool {
		return citizen.Children[i].TotalEnergy() > citizen.Children[j].TotalEnergy()
	})
	for _, child := range citizen.Children {
		if amount == 0 {
			break
		}
		toSurvive := createBabyEnergy - int64(child.HasEnergy)
		if toSurvive <= 0 {
			continue
		}
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &child.Person, "child-preservation"})
		amount -= eatAmount
	}

	if amount > 0 {
		result = append(result, Action{ActionTypeEat, amount, &citizen.Person, "reserving"})
	}

	return result
}

type strategyTrustOnlyOnce struct{}

func (strategy *strategyTrustOnlyOnce) HandleFood(
	citizen *Citizen,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - int64(citizen.HasEnergy)
	if toSurvive > 0 {
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &citizen.Person, "self-preservation"})
		amount -= eatAmount
	}
	sort.Slice(citizen.Children, func(i, j int) bool {
		return citizen.Children[i].TotalEnergy() > citizen.Children[j].TotalEnergy()
	})
	for _, child := range citizen.Children {
		if amount == 0 {
			break
		}
		toSurvive := createBabyEnergy - int64(child.HasEnergy)
		if toSurvive <= 0 {
			continue
		}
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &child.Person, "child-preservation"})
		amount -= eatAmount
	}

	// save those who we can save

	var candidates []*Person
	for _, candidate := range citizen.Playground.People() {
		hasEnergy := candidate.TotalEnergy()
		if hasEnergy < requiredEnergy {
			candidates = append(candidates, candidate)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		totalEnergyI := candidates[i].TotalEnergy()
		totalEnergyJ := candidates[j].TotalEnergy()
		return totalEnergyI > totalEnergyJ
	})
	for _, candidate := range candidates {
		if amount == 0 {
			break
		}
		if candidate.Citizen.SpottedAsGreedyOnce {
			continue
		}
		hasEnergy := candidate.TotalEnergy()
		toSurvive := requiredEnergy - hasEnergy
		if toSurvive > amount {
			toSurvive = amount
		}
		result = append(result, Action{ActionTypeEat, toSurvive, candidate, "altruism"})
		amount -= toSurvive
		if amount == 0 {
			break
		}
	}

	// eat the rest

	if amount > 0 {
		result = append(result, Action{ActionTypeEat, amount, &citizen.Person, "reserving"})
	}
	return result
}

type strategyTrustMirror struct{}

func (strategy *strategyTrustMirror) HandleFood(
	citizen *Citizen,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - int64(citizen.HasEnergy)
	if toSurvive > 0 {
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &citizen.Person, "self-preservation"})
		amount -= eatAmount
	}
	sort.Slice(citizen.Children, func(i, j int) bool {
		return citizen.Children[i].TotalEnergy() > citizen.Children[j].TotalEnergy()
	})
	for _, child := range citizen.Children {
		if amount == 0 {
			break
		}
		toSurvive := createBabyEnergy - int64(child.HasEnergy)
		if toSurvive <= 0 {
			continue
		}
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &child.Person, "child-preservation"})
		amount -= eatAmount
	}

	// save those who we can save

	var candidates []*Person
	for _, candidate := range citizen.Playground.People() {
		hasEnergy := candidate.TotalEnergy()
		if hasEnergy < requiredEnergy {
			candidates = append(candidates, candidate)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		totalEnergyI := candidates[i].TotalEnergy()
		totalEnergyJ := candidates[j].TotalEnergy()
		return totalEnergyI > totalEnergyJ
	})
	for _, candidate := range candidates {
		if amount == 0 {
			break
		}
		if candidate.Citizen.SavedPeople < candidate.Citizen.WasSavedTimes {
			continue
		}
		hasEnergy := candidate.TotalEnergy()
		toSurvive := requiredEnergy - hasEnergy
		if toSurvive > amount {
			toSurvive = amount
		}
		result = append(result, Action{ActionTypeEat, toSurvive, candidate, "altruism"})
		amount -= toSurvive
		if amount == 0 {
			break
		}
	}

	// eat the rest

	if amount > 0 {
		result = append(result, Action{ActionTypeEat, amount, &citizen.Person, "reserving"})
	}
	return result
}

type strategyTrustKindMirror struct{}

func (strategy *strategyTrustKindMirror) HandleFood(
	citizen *Citizen,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - int64(citizen.HasEnergy)
	if toSurvive > 0 {
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &citizen.Person, "self-preservation"})
		amount -= eatAmount
	}
	sort.Slice(citizen.Children, func(i, j int) bool {
		return citizen.Children[i].TotalEnergy() > citizen.Children[j].TotalEnergy()
	})
	for _, child := range citizen.Children {
		if amount == 0 {
			break
		}
		toSurvive := createBabyEnergy - int64(child.HasEnergy)
		if toSurvive <= 0 {
			continue
		}
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &child.Person, "child-preservation"})
		amount -= eatAmount
	}

	// save those who we can save

	var candidates []*Person
	for _, candidate := range citizen.Playground.People() {
		hasEnergy := candidate.TotalEnergy()
		if hasEnergy < requiredEnergy {
			candidates = append(candidates, candidate)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		totalEnergyI := candidates[i].TotalEnergy()
		totalEnergyJ := candidates[j].TotalEnergy()
		return totalEnergyI > totalEnergyJ
	})
	for _, candidate := range candidates {
		if amount == 0 {
			break
		}
		if candidate.Citizen.SavedPeople*2 < candidate.Citizen.WasSavedTimes {
			continue
		}
		hasEnergy := candidate.TotalEnergy()
		toSurvive := requiredEnergy - hasEnergy
		if toSurvive > amount {
			toSurvive = amount
		}
		result = append(result, Action{ActionTypeEat, toSurvive, candidate, "altruism"})
		amount -= toSurvive
		if amount == 0 {
			break
		}
	}

	// eat the rest

	if amount > 0 {
		result = append(result, Action{ActionTypeEat, amount, &citizen.Person, "reserving"})
	}
	return result
}

type strategyTrustEveryGoodTime struct{}

func (strategy *strategyTrustEveryGoodTime) HandleFood(
	citizen *Citizen,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - int64(citizen.HasEnergy)
	if toSurvive > 0 {
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &citizen.Person, "self-preservation"})
		amount -= eatAmount
	}
	sort.Slice(citizen.Children, func(i, j int) bool {
		return citizen.Children[i].TotalEnergy() > citizen.Children[j].TotalEnergy()
	})
	for _, child := range citizen.Children {
		if amount == 0 {
			break
		}
		toSurvive := createBabyEnergy - int64(child.HasEnergy)
		if toSurvive <= 0 {
			continue
		}
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &child.Person, "child-preservation"})
		amount -= eatAmount
	}

	// save those who we can save

	var candidates []*Person
	for _, candidate := range citizen.Playground.People() {
		hasEnergy := candidate.TotalEnergy()
		if hasEnergy < requiredEnergy {
			candidates = append(candidates, candidate)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		totalEnergyI := candidates[i].TotalEnergy()
		totalEnergyJ := candidates[j].TotalEnergy()
		return totalEnergyI > totalEnergyJ
	})
	for _, candidate := range candidates {
		if amount == 0 {
			break
		}
		if candidate.Citizen.SpottedAsGreedyLastTime {
			continue
		}
		hasEnergy := candidate.TotalEnergy()
		toSurvive := requiredEnergy - hasEnergy
		if toSurvive > amount {
			toSurvive = amount
		}
		result = append(result, Action{ActionTypeEat, toSurvive, candidate, "altruism"})
		amount -= toSurvive
		if amount == 0 {
			break
		}
	}

	// eat the rest

	if amount > 0 {
		result = append(result, Action{ActionTypeEat, amount, &citizen.Person, "reserving"})
	}
	return result
}

type strategyTrustAlways struct{}

func (strategy *strategyTrustAlways) HandleFood(
	citizen *Citizen,
	food *Food,
) []Action {
	amount := food.Amount

	var result []Action
	toSurvive := requiredEnergy - int64(citizen.HasEnergy)
	if toSurvive > 0 {
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &citizen.Person, "self-preservation"})
		amount -= eatAmount
	}
	sort.Slice(citizen.Children, func(i, j int) bool {
		return citizen.Children[i].TotalEnergy() > citizen.Children[j].TotalEnergy()
	})
	for _, child := range citizen.Children {
		if amount == 0 {
			break
		}
		toSurvive := createBabyEnergy - int64(child.HasEnergy)
		if toSurvive <= 0 {
			continue
		}
		eatAmount := uint(toSurvive)
		if eatAmount > amount {
			eatAmount = amount
		}
		result = append(result, Action{ActionTypeEat, eatAmount, &child.Person, "child-preservation"})
		amount -= eatAmount
	}

	// save those who we can save

	var candidates []*Person
	for _, candidate := range citizen.Playground.People() {
		hasEnergy := candidate.TotalEnergy()
		if hasEnergy < requiredEnergy {
			candidates = append(candidates, candidate)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		totalEnergyI := candidates[i].TotalEnergy()
		totalEnergyJ := candidates[j].TotalEnergy()
		return totalEnergyI > totalEnergyJ
	})
	for _, candidate := range candidates {
		if amount == 0 {
			break
		}
		hasEnergy := candidate.TotalEnergy()
		toSurvive := requiredEnergy - hasEnergy
		if toSurvive > amount {
			toSurvive = amount
		}
		result = append(result, Action{ActionTypeEat, toSurvive, candidate, "altruism"})
		amount -= toSurvive
		if amount == 0 {
			break
		}
	}

	// hide the rest

	if amount > 0 {
		result = append(result, Action{ActionTypeEat, amount, &citizen.Person, "reserving"})
	}
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
	Destination *Person
	Comment string
}

type Strategy interface {
	HandleFood(citizen *Citizen, food *Food) []Action
}

type Person struct {
	AgeInWeeks uint
	HadEat                    uint
	HasEnergy                 uint
	Playground *Playground
	OwnsFood                  uint
	Citizen *Citizen
}

//go:nosplit
func (person *Person) EatEnergy() uint {
	return person.HadEat
	hadEat := person.HadEat
	if hadEat <= 1000 {
		return hadEat
	}
	return uint(float64(requiredEnergy) + float64(hadEat - requiredEnergy) * extraFoodEfficiency)
}

//go:nosplit
func (person *Person) TotalEnergy() uint {
	return person.HasEnergy + person.OwnsFood + person.EatEnergy()
}

type Child struct {
	Person
	Parent *Citizen
}

func (child *Child) Die() {
	child.Parent.removeChild(child)
}

func (child *Child) Graduate() {
	child.Parent.removeChild(child)
	child.Playground.AddCitizen(child.Parent.Strategy, child.AgeInWeeks)
}

type Citizen struct {
	Person
	Children                  []*Child
	Strategy                  Strategy
	SpottedAsGreedyOnce       bool
	SpottedAsGreedyLastTime   bool
	SavedPeople               uint
	WasSavedTimes             uint
	ChangeStrategyProbability float64
}

func (citizen *Citizen) removeChild(child *Child) {
	for childIdx, childCmp := range child.Parent.Children {
		if childCmp != child {
			continue
		}
		child.Parent.Children[childIdx] = child.Parent.Children[len(child.Parent.Children)-1]
		child.Parent.Children = child.Parent.Children[:len(child.Parent.Children)-1]
		break
	}
}

func (citizen *Citizen) HandleFood(food *Food) []Action {
	return citizen.Strategy.HandleFood(citizen, food)
}

func (citizen *Citizen) CreateBaby() {
	citizen.HasEnergy -= createBabyEnergy
	citizen.Children = append(citizen.Children, &Child{
		Person: Person{
			AgeInWeeks: 0,
			Playground: citizen.Playground,
			Citizen: citizen,
			HasEnergy: createBabyEnergy/2,
		},
		Parent: citizen,
	})
}

type Playground struct {
	Citizens []*Citizen
	weekID uint
	peopleCacheWeekID uint
	peopleCache []*Person
}

func (playground *Playground) people() []*Person {
	var result []*Person
	for _, citizen := range playground.Citizens {
		result = append(result, &citizen.Person)
		for _, child := range citizen.Children {
			result = append(result, &child.Person)
		}
	}
	return result
}

func (playground *Playground) People() []*Person {
	if playground.peopleCacheWeekID != playground.weekID {
		playground.peopleCache = playground.people()
		playground.peopleCacheWeekID = playground.weekID
	}
	return playground.peopleCache
}

func randUintn(n uint) uint {
	return uint(mathrand.ReduceUint64(prng.Uint64AddRotateMultiply(), uint64(n)))
}

func (playground *Playground) AddCitizens(strategy Strategy, citizenAmount uint) {
	for i := uint(0); i < citizenAmount; i++ {
		playground.AddCitizen(strategy, 16*54 + randUintn((80-16)*54))
	}
}

func (playground *Playground) AddCitizen(strategy Strategy, ageInWeeks uint) {
	citizen := &Citizen{
		Strategy:                  strategy,
		ChangeStrategyProbability: rand.Float64()*rand.Float64()*rand.Float64()*rand.Float64(),
	}
	citizen.Person = Person{
		AgeInWeeks: ageInWeeks,
		Playground: playground,
		Citizen: citizen,
	}
	playground.Citizens = append(playground.Citizens, citizen)
}

func (playground *Playground) RemoveCitizen(removeCitizen *Citizen) {
	//fmt.Printf("%#+v\n", removeCitizen)
	for citizenIdx, citizen := range playground.Citizens {
		if citizen != removeCitizen {
			continue
		}

		playground.Citizens[citizenIdx] = playground.Citizens[len(playground.Citizens)-1]
		playground.Citizens = playground.Citizens[:len(playground.Citizens)-1]
	}
}

func (playground *Playground) HasHungryCitizens() bool {
	for _, citizen := range playground.Citizens {
		if citizen.TotalEnergy() < requiredEnergy {
			return true
		}
	}
	return false
}

func (playground *Playground) HungryCitizens() []*Citizen {
	var result []*Citizen
	for _, citizen := range playground.Citizens {
		if citizen.TotalEnergy() < requiredEnergy {
			result = append(result, citizen)
		}
	}
	return result
}

func (playground *Playground) IterateWeek() {
	playground.weekID++

	var foundFood []*Food
	for i := 0; i < amountOfPortions; i++ {
		foundFood = append(foundFood, &Food{false, portionEnergy})
	}

	newCitizenFood := make([][]*Food, len(playground.Citizens))
	for citizenIdx, citizen := range playground.Citizens {
		if citizen.OwnsFood == 0 {
			continue
		}
		newCitizenFood[citizenIdx] = append(newCitizenFood[citizenIdx], &Food{true, citizen.OwnsFood})
		citizen.OwnsFood = 0
	}

	for len(foundFood) > 0 {
		if len(playground.Citizens) == 0 {
			return
		}
		for _, foodPortion := range foundFood {
			citizenIdx := randUintn(uint(len(playground.Citizens)))
			newCitizenFood[citizenIdx] = append(newCitizenFood[citizenIdx], foodPortion)
		}
		foundFood = foundFood[:0]

		shuffler.Shuffle(len(playground.Citizens), func(i, j int) {
			playground.Citizens[i], playground.Citizens[j] = playground.Citizens[j], playground.Citizens[i]
		})
		for citizenIdx, citizen := range playground.Citizens {
			newFood := newCitizenFood[citizenIdx]
			newCitizenFood[citizenIdx] = newCitizenFood[citizenIdx][:0]

			for foodIdx := range newFood {
				foodPortion := newFood[len(newFood) - foodIdx-1] // first we handle non-hidden food

				isGreedy := false
				if citizen.TotalEnergy()+foodPortion.Amount > requiredEnergy && playground.HasHungryCitizens() {
					// opportunity for altruism
					isGreedy = true
				}

				actions := citizen.HandleFood(foodPortion)
				usedFood := uint(0)
				for _, action := range actions {
					if action.Amount == 0 {
						panic(fmt.Sprintf("action.Amount == 0: %#+v %#+v", citizen, action))
					}
					if action.Amount > foodPortion.Amount-usedFood {
						panic(fmt.Sprintf("cheater! %+v: %d > %+v - %d (%T)",
							action, action.Amount, foodPortion, usedFood, citizen.Strategy))
					}
					if action.Destination.Citizen != citizen { // altruism
						if action.Destination.TotalEnergy() < requiredEnergy &&
							action.Destination.TotalEnergy() + action.Amount >= requiredEnergy {
							citizen.SavedPeople++
							action.Destination.Citizen.WasSavedTimes++
							isGreedy = false
						}
					}
					usedFood += action.Amount
					switch action.ActionType {
					case ActionTypeEat:
						action.Destination.HadEat += action.Amount
					case ActionTypeHide:
						action.Destination.OwnsFood += action.Amount
					default:
						panic("unknown action")
					}
				}
				if usedFood != foodPortion.Amount {
					panic(fmt.Sprintf("something is wrong: %d != %+v (%T)",
						usedFood, foodPortion, citizen.Strategy))
				}
				if citizen.TotalEnergy() < requiredEnergy &&
					citizen.TotalEnergy() - citizen.HadEat + foodPortion.Amount >= requiredEnergy {
					panic(fmt.Sprintf("suicide strategy: 0x%p:%#+v %#+v %#+v %v",
						citizen, citizen, foodPortion, actions, citizen.TotalEnergy()))
				}

				citizen.SpottedAsGreedyOnce = citizen.SpottedAsGreedyOnce || isGreedy
				citizen.SpottedAsGreedyLastTime = isGreedy
			}
		}
	}

	// Dying from hunger
	for _, citizen := range playground.Citizens {
		for _, child := range citizen.Children {
			child.HasEnergy += child.EatEnergy()
			child.HadEat = 0
			if child.TotalEnergy() < requiredEnergy {
				child.Die()
				continue
			}
			child.HasEnergy -= requiredEnergy
		}
		citizen.HasEnergy += citizen.EatEnergy()
		citizen.HadEat = 0
		if citizen.HasEnergy < requiredEnergy {
			if citizen.TotalEnergy() >= requiredEnergy {
				panic(fmt.Sprintf("invalid strategy: %+v", citizen))
			}
			citizen.Playground.RemoveCitizen(citizen)
			continue
		}
		citizen.HasEnergy -= requiredEnergy
	}

	// Generate babies
	if enableChildren {
		for _, citizen := range playground.Citizens {
			alreadyHasUnbornBaby := false
			for _, child := range citizen.Children {
				if child.AgeInWeeks < 40 {
					alreadyHasUnbornBaby = true
				}
			}
			if alreadyHasUnbornBaby {
				continue
			}
			if citizen.HasEnergy >= startBabyEnergy {
				citizen.CreateBaby()
			}
		}
	}

	// Aging
	if enableAging {
		for _, citizen := range playground.Citizens {
			for _, child := range citizen.Children {
				child.AgeInWeeks++
			}
			citizen.AgeInWeeks++
			if citizen.AgeInWeeks > personExpirationInWeeks {
				citizen.Playground.RemoveCitizen(citizen)
			}
		}
	}

	// Graduation
	if enableChildren {
		for _, citizen := range playground.Citizens {
			for _, child := range citizen.Children {
				if child.AgeInWeeks > personGraduationInWeeks {
					child.Graduate()
				}
			}
		}
	}

	// New strategies
	nextStrategy := make([]Strategy, len(playground.Citizens))
	for citizenIdx, citizen := range playground.Citizens {
		if rand.Float64() < citizen.ChangeStrategyProbability {
			nextStrategy[citizenIdx] = playground.Citizens[randUintn(uint(len(playground.Citizens)))].Strategy
		}
	}
	for citizenIdx, strategy := range nextStrategy {
		if strategy == nil {
			continue
		}
		playground.Citizens[citizenIdx].Strategy = strategy
	}
}

func main() {

	go func() {
		log.Fatal(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	allStrategies := []Strategy{
		&strategyDoNotTrust{},
		&strategyTrustOnlyOnce{},
		&strategyTrustMirror{},
		&strategyTrustKindMirror{},
		&strategyTrustEveryGoodTime{},
		&strategyTrustAlways{},
	}


	for _, strategy := range allStrategies {
		strategies := []Strategy{&strategyDoNotTrust{}, strategy}
		//strategies := allStrategies
		totalPopulation := make([]uint64, len(allStrategies))
		populationByFlexibility := make([]uint, 10)
		populationByFlexibilitySurvived := make([]uint, 10)
		var noPopulation uint


		var wg sync.WaitGroup
		var mutex sync.Mutex
		for i := 0; i < tries; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				playground := &Playground{}
				for _, strategy := range strategies {
					playground.AddCitizens(strategy, familySize)
				}

				mutex.Lock()
				for _, citizen := range playground.Citizens {
					populationByFlexibility[uint(citizen.ChangeStrategyProbability * 10)]++
				}
				mutex.Unlock()

				for week := 0; week < 200 * 54; /* 200 years */ week++ {
					if i == 0 && false {
						localPopulation := make([]uint64, len(allStrategies))
						for _, citizen := range playground.Citizens {
							for strategyIdx, strategy := range allStrategies {
								if strategy != citizen.Strategy {
									continue
								}
								localPopulation[strategyIdx]++
							}
						}
						fmt.Println(week, localPopulation, len(playground.Citizens), i)
					}
					playground.IterateWeek()
				}

				mutex.Lock()
				for _, citizen := range playground.Citizens {
					populationByFlexibilitySurvived[uint(citizen.ChangeStrategyProbability * 10)]++
				}
				mutex.Unlock()

				mutex.Lock()
				for _, citizen := range playground.Citizens {
					for strategyIdx, strategy := range allStrategies {
						if strategy != citizen.Strategy {
							continue
						}
						totalPopulation[strategyIdx]++
					}
				}
				if len(playground.Citizens) == 0 {
					noPopulation++
				}
				mutex.Unlock()
			}(i)
		}
		wg.Wait()

		for strategyIdx := 0; strategyIdx<len(totalPopulation); strategyIdx++ {
			survived := totalPopulation[strategyIdx]
			fmt.Printf("strategy #%d: sum of survived in %d tries: %d. Growth rate: %.2f%%\n",
				strategyIdx+1, tries, survived, float64(survived)/float64(tries)/familySize*100 - 100)
		}

		for idx, survived := range populationByFlexibilitySurvived {
			fmt.Printf("change strategy rate range [%.1f-%.1f): Growth rate: %.2f%%\n",
				float64(idx)/10, float64(idx+1)/10, float64(survived)/float64(populationByFlexibility[idx])*100 - 100)
		}

		fmt.Printf("genocide rate: %.2f%%\n", float64(noPopulation) / tries * 100)
	}
}
