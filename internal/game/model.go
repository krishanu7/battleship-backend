package game

import (
	"fmt"
	"strings"
)

type ShipType string

const (
	Carrier    ShipType = "Carrier"
	Battleship ShipType = "Battleship"
	Cruiser    ShipType = "Cruiser"
	Submarine  ShipType = "Submarine"
	Destroyer  ShipType = "Destroyer"
)

type Ship struct {
	Type        ShipType `json:"type"`
	Size        int
	Start       string   `json:"start"`
	Orientation string   `json:"orientation"`
	Cells       []string
}

type Board struct {
	PlayerID string
	RoomID   string
	Ships    []Ship
	Grid     map[string]string // {"A1": "Carrier", "A2": "Carrier", ...}
}

var ShipConfig = map[ShipType]int{
	Carrier:    5,
	Battleship: 4,
	Cruiser:    3,
	Submarine:  3,
	Destroyer:  2,
}

// converts "A1" to row (0-9) and col (0-9).
func ParseCoordinate(coord string) (row, col int, err error) {
	if len(coord) < 2 {
		return 0, 0, fmt.Errorf("invalid coordinate: %s", coord)
	}
	rowChar := strings.ToUpper(string(coord[0]))
	colStr := coord[1:]

	if rowChar < "A" || rowChar > "J" {
		return 0, 0, fmt.Errorf("invalid row: %s", rowChar)
	}
	colNum, err := fmt.Sscanf(colStr, "%d", &col)
	if err != nil || colNum != 1 {
		return 0, 0, fmt.Errorf("invalid column: %s", colStr)
	}
	if col < 1 || col > 10 {
		return 0, 0, fmt.Errorf("column out of bounds: %d", col)
	}	
	return int(rowChar[0] - 'A'), col - 1, nil
}

//converts row (0-9) and col (0-9) to "A1"
func FormatCoordinate(row, col int) string {
	return fmt.Sprintf("%c%d", 'A'+row, col+1)
}