package common

import (
	"fmt"
	"os"
	"encoding/csv"
)

type BetInfo struct {
	Agency    string
	Name      string
	Surname   string
	Document  string
	Birthdate string
	Number    string
}
// opens the .csv file and reads it, parsing every line into an instance of BetInfo 
func loadBetsFromFile(agencyID string) ([]BetInfo, error) {
	file, err := os.Open("agency.csv")
	if err != nil {
		log.Errorf("action: read_bet_file | result: fail | error: %v", err)
		return nil, err
	}
	reader := csv.NewReader(file)
	lines, err := reader.ReadAll()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("Error reading file: %v", err)
	}
	var bets []BetInfo 
	for _, line := range lines {
		bet := BetInfo {
			Agency: agencyID,
			Name: line[0],
			Surname: line[1],
			Document: line[2],
			Birthdate: line[3],
			Number: line[4],
		}
		bets = append(bets, bet)
	}
	file.Close()
	return bets, nil
}
