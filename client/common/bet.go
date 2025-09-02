package common

import (
	"fmt"
	"os"
)

type BetInfo struct {
	Agency    string
	Name      string
	Surname   string
	Document  string
	Birthdate string
	Number    string
}

type ServerResponse struct {
	Type    string
	Status  string
	Message string
}

func loadBetDataFromEnv(clientID string) (BetInfo, error) {
	var bet BetInfo

	bet.Agency = clientID
	bet.Name = os.Getenv("NAME")
	bet.Surname = os.Getenv("SURNAME")
	bet.Document = os.Getenv("DOCUMENT")
	bet.Birthdate = os.Getenv("BIRTHDATE")

	numberStr := os.Getenv("NUMBER")

	if numberStr == "" {
		return bet, fmt.Errorf("NUMBER environment variable is required")
	}

	bet.Number = numberStr

	if bet.Name == "" || bet.Surname == "" || bet.Document == "" || bet.Birthdate == "" {
		return BetInfo{}, fmt.Errorf("All bet data fields are required")
	}

	return bet, nil
}
