package main

import (
	"fmt"
)

type FooEntity interface {
	getID() int
}

type BarEntity interface {
	getName() string
}

type TestPlayer struct {
	pID   *int
	pName *string
}

func (player TestPlayer) getID() int {
	// Implementing `FooEntity`.
	var ID int
	ID = *(player.pID)
	return ID
}

func (pPlayer *TestPlayer) getName() string {
	// Implementing `BarEntity`.
	/**
	 * NOTE: This kind of method is callable by a `player TestPlayer` instance as well.
	 */
	var player TestPlayer
	player = *pPlayer
	var name string
	name = *(player.pName)
	return name
}

func printIDOfFooEntityByCastingToStruct(e FooEntity) {
	// Reference https://tour.golang.org/methods/15.
	testPlayerIns := e.(TestPlayer)
	var ID int
	ID = *(testPlayerIns.pID)
	fmt.Printf("%d\n", ID)
}

func printNameOfBarEntityByCastingToPtrToStruct(e BarEntity) {
	pTestPlayerIns := e.(*TestPlayer)
	testPlayerIns := *pTestPlayerIns
	var name string
	name = *(testPlayerIns.pName)
	fmt.Printf("%s\n", name)
}

func main() {
	ID := 1
	pID := &ID
	Name := "Tom"
	pName := &Name
	player := TestPlayer{
		pID:   pID,
		pName: pName,
	}
	/**
	 * // WARNING: Invalid syntax.
	 * fmt.Printf("%v\n", player.(*pName))
	 */

	// fmt.Printf("Player name is %v at HeapRAM addr = %v.\n", *(player.pName), player.pName)
	fmt.Printf("Player ID is %d at HeapRAM addr = %p.\n", player.getID(), player.pID)

	var pPlayer *TestPlayer
	pPlayer = (&player)
	fmt.Printf("Player name is %s at HeapRAM addr = %p.\n", pPlayer.getName(), player.pName)

	// WARNING: Also valid syntax.
	fmt.Printf("\n[Alternative call to interface BarEntity]\nPlayer ID is %v at HeapRAM addr = %v.\nPlayer name is %v at HeapRAM addr = %v.\n\n", player.getID(), player.pID, player.getName(), player.pName)

	printIDOfFooEntityByCastingToStruct(player)
	printNameOfBarEntityByCastingToPtrToStruct(pPlayer)
}
