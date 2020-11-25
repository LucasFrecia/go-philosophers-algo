package main

import (
	"fmt"
	"sync"
)

/*
	Implement the dining philosopher’s problem with the following constraints/modifications.
		1. There should be 5 philosophers sharing chopsticks, with one chopstick between each adjacent pair of philosophers.
		2. Each philosopher should eat only 3 times
		3. The philosophers pick up the chopsticks lowest-numbered first
		4. 2 philosophers can eat concurrently.
		5. Each philosopher is numbered, 1 through 5.
		6. When a philosopher starts eating (after it has obtained necessary locks) it prints “starting to eat <number>” on a line by itself,
		   where <number> is the number of the philosopher.
		7. When a philosopher finishes eating (before it has released its locks) it prints “finishing eating <number>” on a line by itself,
		   where <number> is the number of the philosopher.

		@author Lucas Frecia
*/

type ChopS struct {
	mu 	    sync.Mutex
	id 	    int
	philosopher *Philo
}

type Philo struct {
	mu 		sync.Mutex
	id 		int
	leftCs, rightCs *ChopS
	ate 		int
	eating 		bool
}

var finishedEatingChannel   chan *Philo
var eatingChannel 	    chan *Philo
var leaveTableChannel 	    chan *Philo
var pickUpChopsticks 	    chan *Philo
var askTheHostToEat 	    chan *Philo
var readyToEat 		    chan *Philo
var returnChopSticksChannel chan *Philo

func (philo *Philo) getChopS(chopS []*ChopS) {
	chopS[philo.id - 1].mu.Lock()
	chopS[(philo.id) % 5].mu.Lock()
	chL := chopS[philo.id - 1].philosopher
	chR := chopS[(philo.id) % 5].philosopher
	chopS[philo.id - 1].mu.Unlock()
	chopS[(philo.id) % 5].mu.Unlock()
	if chL == nil && chR == nil {
		philo.pickUpCS(chopS[philo.id - 1], "right")
		philo.pickUpCS(chopS[(philo.id) % 5], "left")
	}

	philo.mu.Lock()
	if  philo.leftCs != nil && philo.rightCs != nil {
		readyToEat <- philo
	} else {
		go philo.sitInTable()
	}
	philo.mu.Unlock()
}

func (philo *Philo) pickUpCS(ch *ChopS, hand string) {
	ch.mu.Lock()
	philo.mu.Lock()
	if hand == "left" {
		philo.leftCs = ch
	} else if hand == "right" {
		philo.rightCs = ch
	}
	ch.philosopher = philo
	philo.mu.Unlock()
	ch.mu.Unlock()
}

func (philo *Philo) dropCS(ch *ChopS, hand string) {
	ch.mu.Lock()
	philo.mu.Lock()
	if hand == "left" {
		philo.leftCs = nil
	} else if hand == "right" {
		philo.rightCs = nil
	}
	ch.philosopher = nil
	philo.mu.Unlock()
	ch.mu.Unlock()
}

func (philo *Philo) sitInTable() {
	pickUpChopsticks <- philo
}

func (philo *Philo) Eat() {
	eatingChannel <- philo
}

func (philo *Philo) finishEating() {
	finishedEatingChannel <- philo
}

func (philo *Philo) leaveTable() {
	leaveTableChannel <- philo
}

func (philo *Philo) tellHostDoneEating() {
	philo.mu.Lock()
	philo.ate = philo.ate + 1
	philo.mu.Unlock()
	returnChopSticksChannel <- philo
}

func (philo *Philo) askHostForPermissiontoEat() {
	if philo.leftCs != nil && philo.rightCs != nil  {
		askTheHostToEat <- philo
	} else {
		returnChopSticksChannel <- philo
	}
}

func main() {
	CSticks := make([]*ChopS, 5)
	Philos  := make([]*Philo, 5)
	finishedEatingChannel = make(chan *Philo)
	eatingChannel = make(chan *Philo)
	leaveTableChannel = make(chan *Philo)
	pickUpChopsticks = make(chan *Philo)
	askTheHostToEat = make(chan *Philo, 1)
	readyToEat = make(chan *Philo)
	returnChopSticksChannel = make(chan *Philo)
	philosophersSittingAtTable := 5

	for i := 0; i < 5; i++ {
		Philos[i] = &Philo{sync.Mutex{}, i + 1, nil, nil, 0, false }
		CSticks[i] = &ChopS{sync.Mutex{}, i + 1, nil }
	}

	for  i := range Philos {
		go Philos[i].sitInTable()
	}

	for {
		select {
		case philosopher := <-pickUpChopsticks:
			go philosopher.getChopS(CSticks)
		case philosopher := <-readyToEat:
			go philosopher.askHostForPermissiontoEat()
		case philosopher := <-askTheHostToEat:
			go philosopher.Eat()
		case philosopher := <-eatingChannel:
			fmt.Printf("starting to eat %d \n", philosopher.id)
			go philosopher.finishEating()
		case philosopher := <-finishedEatingChannel:
			fmt.Printf("finishing eating %d \n", philosopher.id)
			go philosopher.tellHostDoneEating()
		case philosopher := <-returnChopSticksChannel:
			if philosopher.rightCs != nil {
				go philosopher.dropCS(philosopher.rightCs, "right")
			}
			if philosopher.leftCs != nil {
				go philosopher.dropCS(philosopher.leftCs, "left")
			}
			if philosopher.ate < 3 {
				go philosopher.sitInTable()
			} else {
				go philosopher.leaveTable()
			}
		case philosopher := <-leaveTableChannel:
			fmt.Printf("philosopher %d left the table after eating %d times \n", philosopher.id, philosopher.ate)
			philosophersSittingAtTable--
			if philosophersSittingAtTable == 0 {
				return
			}
		}
	}
}
