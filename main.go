package main

import (
	"container/heap"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func UnixtimeMilli() int64 {
	return time.Now().UnixNano() / 1000000
}

type Player struct {
	CreatedAt int64  `json:"createdAt"`
	ID        int    `json:"id"`
	Name      string `json:"name"`
}

type roomState struct {
	IDLE          int
	WAITING       int
	IN_BATTLE     int
	IN_SETTLEMENT int
	IN_DISMISSAL  int
}

// A single instance containing only "named constant integers" to be shared by all threads.
var RoomState *roomState

func calRoomScore(playerListCount int, capacity int, currentRoomState int) float32 {
	x := float32(playerListCount) / float32(capacity)
	d := (x - 0.2)
	d2 := d * d
	return -7.8125*d2 + 5 - float32(currentRoomState)
}

type Room struct {
	ID       int             `json:"id"`
	Capacity int             `json:"capacity"`
	Players  map[int]*Player `json:"players"`
	Score    float32
	State    int
	Index    int
}

func (pR *Room) updateScore() {
	pR.Score = calRoomScore(len(pR.Players), pR.Capacity, pR.State)
}

func (pR *Room) addPlayerIfPossible(pPlayer *Player) bool {
	// TODO: Check feasibility first.
	pR.Players[pPlayer.ID] = pPlayer
	pR.updateScore()
	return true
}

var RoomHeapMux sync.Mutex

// Reference https://golang.org/pkg/container/heap/.
type RoomHeap []Room

func (pPq *RoomHeap) PrintInOrder() {
	pq := *pPq
	fmt.Printf("The RoomHeap instance now contains:\n")
	for i := 0; i < len(pq); i++ {
		fmt.Printf("{index: %d, roomID: %d, score: %.2f} ", i, pq[i].ID, pq[i].Score)
	}
	fmt.Printf("\n")
}

/*
Note that using `[]*Room` takes extra RAM for storing each "*Room", but could help postpone the RAM allocation of actual "Room" instance. We don't need this advantage in the current example.

To be quantitative, `make([]*Room, 1024)` immediately takes 1024*32_bits/ptr, and `1024*(32_bits/ptr + sizeof(Room)_bits/ptr)` at most if all instantiated.

In contrast, `make([]Room, 1024)` immediately takes 1024*sizeof(Room)_bits/ptr, but won't grow with later assignment.

This is why we're having `Room.Players map[int]*Player` here.
*/

func (pq RoomHeap) Len() int { return len(pq) }

func (pq RoomHeap) Less(i, j int) bool {
	return pq[i].Score > pq[j].Score
}

func (pq *RoomHeap) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].Index = i
	(*pq)[j].Index = j
}

func (pq *RoomHeap) Push(pItem interface{}) {
	// NOTE: Must take input param type `*Room` here.
	n := len(*pq)
	pItem.(*Room).Index = n
	*pq = append(*pq, *(pItem.(*Room)))
}

func (pq *RoomHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	if n == 0 {
		panic(fmt.Sprintf("Popping on an empty heap is not allowed.\n"))
	}
	item := old[n-1]
	if item.Score <= float32(0.0) {
		panic(fmt.Sprintf("No available room at the moment.\n"))
	}
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	// NOTE: Must return instance which is directly castable to type `*Room` here.
	return (&item)
}

func (pq *RoomHeap) update(pItem *Room, Score float32) {
	// NOTE: Must use type `*Room` here.
	heap.Fix(pq, pItem.Index)
}

func main() {
	// Init "pseudo class constants".
	RoomState = &roomState{
		IDLE:          0,
		WAITING:       0,
		IN_BATTLE:     9999999,
		IN_SETTLEMENT: 9999999,
		IN_DISMISSAL:  9999999,
	}

	initialCountOfRooms := 5
	pq := make(RoomHeap, initialCountOfRooms)

	roomCapacity := 4

	for i := 0; i < initialCountOfRooms; i++ {
		players := make(map[int]*Player)
		currentRoomState := RoomState.IDLE
		pq[i] = Room{
			Players:  players,
			Capacity: roomCapacity,
			Score:    calRoomScore(len(players) /* Initially 0. */, roomCapacity, currentRoomState),
			State:    currentRoomState,
			ID:       i,
			Index:    i,
		}
	}
	heap.Init(&pq)
	fmt.Printf("RoomHeap is initialized.\n")

	var wasteChanWg sync.WaitGroup

	/**
	 * The `wasteChanList` and `wasteChanListCloseSignalChan` together shows an approach to gracefully terminate a goroutine which
	 * waits indefinitely for I/O of a channel.
	 *
	 * Note that `wasteChanListCloseSignalChan` need NOT be closed explicitly.
	 */
	aWasteChan := make(chan interface{}, 1024)
	anotherWasteChan := make(chan interface{}, 1024)
	wasteChanList := make([]chan interface{}, 2)

	wasteChanList[0] = aWasteChan
	wasteChanList[1] = anotherWasteChan

	wasteChanListCloseSignalChan := make(chan bool, 1)

	wasteChanLoopFPS := 10
	wasteChanLoopMillisPerFrame := 1000 / wasteChanLoopFPS
	wasteChanLoop := func(wasteChan chan interface{}, loopName string) error {
		defer func() {
			wasteChanWg.Done()
			fmt.Printf("The loop for `%s` is ended.\n", loopName)
		}()
		for {
			select {
			case _ = <-wasteChan:
			default:
			}
			select {
			case trueOrFalse := <-wasteChanListCloseSignalChan:
				fmt.Printf("Received %v from wasteChanListCloseSignalChan in loop for `%s`.\n", trueOrFalse, loopName)
				if trueOrFalse == true {
					return nil
				}
			default:
			}
			time.Sleep(time.Millisecond * time.Duration(wasteChanLoopMillisPerFrame))
		}
		return nil
	}
	wasteChanWg.Add(len(wasteChanList))
	go wasteChanLoop(aWasteChan, "aWasteChan")
	go wasteChanLoop(anotherWasteChan, "anotherWasteChan")

	var mainWg sync.WaitGroup
	initialCountOfPlayers := 100
	mainWg.Add(initialCountOfPlayers)
	for i := 0; i < initialCountOfPlayers; i++ {
		innerNow := UnixtimeMilli()
		testingPlayer := Player{
			CreatedAt: innerNow,
			ID:        i,
			Name:      fmt.Sprintf("Player#%d", i),
		}
		fmt.Printf("Has generated player %v at %v.\n", testingPlayer.Name, testingPlayer.CreatedAt)
		go func(tPlyr *Player) {
			defer mainWg.Done()
			// It's possible yet not recommended to acquire the current "Goroutine ID" for printing. Search for "Goroutine ID" for more information.
			randomMillisToSleep := rand.Intn(100) // [0, 100) milliseconds.
			time.Sleep(time.Duration(randomMillisToSleep) * time.Millisecond)

			RoomHeapMux.Lock()
			defer RoomHeapMux.Unlock()
			defer func() {
				// Will immediately execute `RoomHeapMux.Unlock()` and then `mainWg.Done()` in order if panics.
				if r := recover(); r != nil {
					fmt.Println("Recovered from a panic: ", r)
				}
			}()
			pRoom := heap.Pop(&pq).(*Room)
			fmt.Printf("Successfully popped room %v for player %v.\n", pRoom.ID, tPlyr.Name)
			randomMillisToSleepAgain := rand.Intn(100) // [0, 100) milliseconds.
			time.Sleep(time.Duration(randomMillisToSleepAgain) * time.Millisecond)
			pRoom.addPlayerIfPossible(tPlyr)
			heap.Push(&pq, pRoom)
			(&pq).update(pRoom, pRoom.Score)
			pq.PrintInOrder()
		}(&testingPlayer)
	}

	now := UnixtimeMilli()
	fmt.Printf("Starting to wait for all `goroutines of room joining` to end at %v.\n", now)
	mainWg.Wait()
	now = UnixtimeMilli()
	fmt.Printf("All `goroutines of room joining` ended at %v.\n", now)
	wasteChanListCloseSignalChan <- true
	wasteChanListCloseSignalChan <- true
	wasteChanWg.Wait()
	now = UnixtimeMilli()
	fmt.Printf("Exiting at %v.\n", now)
}
