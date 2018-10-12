package main

import (
	"container/heap"
  "math/rand"
	"fmt"
  "time"
  "sync"
)

func UnixtimeMilli() int64 {
  return time.Now().UnixNano()/1000000
}

type Player struct {
	CreatedAt     int64      `json:"createdAt"`
	ID            int        `json:"id"`
	Name          string     `json:"name"`
}

type roomState struct {
  IDLE  int
  WAITING int
  IN_BATTLE int
  IN_SETTLEMENT int
  IN_DISMISSAL int
}

// A single instance containing only "named constant integers" to be shared by all threads. 
var RoomState *roomState

func calRoomScore(playerListCount int, capacity int, currentRoomState int) float32 {
  x := float32(playerListCount)/float32(capacity)
  return -7.8125*(x - 0.2) + 5 - float32(currentRoomState)
}

type Room struct {
	ID            int        `json:"id"`
  Capacity      int        `json:"capacity"`
  Players       map[int]*Player  `json:"players"`
  Score         float32
  State         int
  Index         int
}

var RoomHeapMux sync.Mutex
// Reference https://golang.org/pkg/container/heap/.
type RoomHeap []Room
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

func (pq RoomHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *RoomHeap) Push(x interface{}) {
	n := len(*pq)
	item := x.(Room)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *RoomHeap) Pop() interface{} {
	old := *pq
	n := len(old)
  if n == 0 {
    panic(fmt.Sprintf("Popping on an empty heap is not allowed.\n"))
  }
	item := old[n-1]
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *RoomHeap) update(item Room, Score float32) {
	item.Score = Score
	heap.Fix(pq, item.Index)
}

func main() {
  // Init "pseudo class constants".
  RoomState = &roomState{
    IDLE: 0,
    WAITING: 0,
    IN_BATTLE: 9999999,
    IN_SETTLEMENT: 9999999,
    IN_DISMISSAL: 9999999,
  }

  initialCountOfRooms := 20
  pq := make(RoomHeap, initialCountOfRooms)

  roomCapacity := 4

  for i := 0; i < initialCountOfRooms; i++ {
    var players map[int]*Player
    currentRoomState := RoomState.IDLE
    pq[i] = Room{
      Players: players,
      Capacity: roomCapacity,
      Score: calRoomScore(len(players) /* Initially 0. */, roomCapacity, currentRoomState),
      State: currentRoomState,
      ID: i,
      Index: i,
    }
  }
  heap.Init(&pq)
  fmt.Printf("RoomHeap is initialized.\n")

  var wg sync.WaitGroup
  initialCountOfPlayers := 100
  wg.Add(initialCountOfPlayers)
  for i:= 0; i < initialCountOfPlayers; i++ {
    innerNow := UnixtimeMilli()
    testingPlayer := Player{
      CreatedAt:  innerNow,
      ID: i,
      Name: fmt.Sprintf("Player#%d", i),
    }
    fmt.Printf("Has generated player %v at %v.\n", testingPlayer.Name, testingPlayer.CreatedAt)
    go func(tPlyr *Player) {
      defer func() {
        if r := recover(); r != nil {
          fmt.Println("Recovered from a panic", r)
        }
      }()
      defer wg.Done()
      // It's possible yet not recommended to acquire the current "Goroutine ID" for printing. Search for "Goroutine ID" for more information.
      RoomHeapMux.Lock()
      defer RoomHeapMux.Unlock()
      room := heap.Pop(&pq).(Room)
      fmt.Printf("Successfully popped room %v for player %v.\n", room.ID, (*tPlyr).Name);
      // Will immediately execute `RoomHeapMux.Unlock()` and then `wg.Done()` in order if panics.
      randomMillisToSleep := rand.Intn(100) // [0, 100) milliseconds.
      time.Sleep(time.Duration(randomMillisToSleep) * time.Millisecond)
    }(&testingPlayer)
  }

  now := UnixtimeMilli()
  fmt.Printf("Starting to wait for all goroutines to end at %v.\n", now)
  wg.Wait()
  now = UnixtimeMilli()
  fmt.Printf("Exiting at %v.\n", now)
}
