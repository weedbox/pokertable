package seat_manager

import "fmt"

func DebugPrintSeats(msg string, sm SeatManager) {
	fmt.Printf("[%s] Dealer: %d, SB: %d, BB: %d\n", msg, sm.CurrentDealerSeatID(), sm.CurrentSBSeatID(), sm.CurrentBBSeatID())
	seats := sm.Seats()
	for i := 0; i < len(seats); i++ {
		seatPlayer := seats[i]
		if seatPlayer == nil {
			fmt.Printf("Seat %d is empty\n", i)
		} else {
			fmt.Printf("Seat %d is occupied by %s. Active: %t\n", i, seatPlayer.ID, seatPlayer.Active)
		}
	}
}
