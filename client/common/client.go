package common

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"
	"strings"
	"os"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type BetInfo struct {
	Agency 		string
	Name 		string
	Surname 	string
	Document 	string
	Birthdate 	string
	Number 		string
}

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
	betInfo BetInfo
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	
	if err := client.loadBetDataFromEnv(); err != nil {
		log.Criticalf("action: load_bet_data | result: fail | client_id: %v | error: %v", config.ID, err)
	}

	return client
}

// loadBetDataFromEnv reads environment variables and creates BetInfo struct
func (c* Client) loadBetDataFromEnv() error {
	c.betInfo.Agency = c.config.ID
	c.betInfo.Name = os.Getenv("NAME")
	c.betInfo.Surname = os.Getenv("SURNAME")
	c.betInfo.Document = os.Getenv("DOCUMENT")
	c.betInfo.Birthdate = os.Getenv("BIRTHDATE")

	numberStr := os.Getenv("NUMBER")
	if numberStr == "" {
		return fmt.Errorf("NUMBER environment variable is required")
	}
	
	c.betInfo.Number = numberStr

	if c.betInfo.Name == "" || c.betInfo.Surname == "" || c.betInfo.Document == "" || c.betInfo.Birthdate == "" {
		return fmt.Errorf("All bet data fields are required")
	}

	return nil
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// createBetMessage serializes BetInfo struct in Client
// Messages are styled: "TYPE/data_1/.../data_n\n"
func (c* Client) createBetMessage() string {
	return fmt.Sprintf("BET/%s/%s/%s/%s/%s/%s\n",
			c.betInfo.Agency,
			c.betInfo.Name,
			c.betInfo.Surname,
			c.betInfo.Document, 
			c.betInfo.Birthdate,
			c.betInfo.Number)
}

// sendBet creates the bet message and sends it through the socket. 
// it also waits for server ack to ensure correct communication flow
func (c* Client) sendBet() error {
	betMessage := c.createBetMessage()

	if err := c.sendMessage(betMessage); err != nil {
		return fmt.Errorf("failed to send bet: %v", err)
	}

	response, err := c.receiveMessage()
	if err != nil {
		return fmt.Errorf("failed to receive response from server: %v", err)
	}

	if err := c.parseResponseFromServer(response); err != nil {
		return fmt.Errorf("failed to parse response from server: %v", err)
	}

	return nil
}

// sendMessage sends a string through the socket, it continues sending until the complete
// message is transmitted, avoiding short-writes
func (c* Client) sendMessage(message string) error {
	data := []byte(message)
	totalSent := 0

	for totalSent < len(data) {
		sent, err := c.conn.Write(data[totalSent:])
		if err != nil {
			return err
		}
		totalSent += sent
	}

	return nil
}

// receiveMessage receives server answer
// TODO: handle short-reads
func (c* Client) receiveMessage() (string, error) {
	reader := bufio.NewReader(c.conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(message), nil
}

// parseResponseFromServer receives the message from the server and verifies its contents
func (c* Client) parseResponseFromServer(response string) error {
	parts := strings.Split(response, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid response format")
	}

	if parts[0] != "RESPONSE" {
		return fmt.Errorf("unexpected response type: %s", parts[0])
	}

	status := parts[1]
	message := parts[2]

	if status == "SUCCESS" {
		log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s",
			c.betInfo.Document, c.betInfo.Number)
		return nil
	} else {
		log.Errorf("action: apuesta_enviada | result: fail | dni: %s | numero: %s | error: %s",
			c.betInfo.Document, c.betInfo.Number, message)
		return fmt.Errorf("server error: %s", message)
	}
}

// closeConnection Close connection with logging
func (c *Client) closeConnection() {
	if c.conn != nil {
		c.conn.Close()
		log.Infof("action: close_connection | result: success | client_id: %v", c.config.ID)
		c.conn = nil
	}
}

// StartClientLoop Send messages to the client until some time threshold is met
	func (c *Client) StartClientLoop(ctx context.Context) {
		select {
		case <-ctx.Done():
			log.Infof("action: shutdown | result: success | client_id: %v", c.config.ID)
				if c.conn != nil {
					c.closeConnection()
				}
				return
		default:
		}

		c.createClientSocket()

		if err := c.sendBet(); err != nil {
			log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}

		c.closeConnection();

		select {
		case <-ctx.Done():
				log.Infof("action: shutdown | result: succes | client_id: %v", c.config.ID)
				return
		case <-time.After(c.config.LoopPeriod):

		}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
