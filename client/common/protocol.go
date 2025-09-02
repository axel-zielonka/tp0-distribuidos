package common

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"encoding/binary"

	"github.com/op/go-logging"
)

var protocolLog = logging.MustGetLogger("protocol")

type ClientProtocol struct {
	conn net.Conn
	id string
}

func NewClientProtocol(id string, conn net.Conn) (ClientProtocol) {
	return ClientProtocol{conn: conn, id: id}
}

func (cp *ClientProtocol) SendBet(bet BetInfo) (*ServerResponse, error) {
	message := cp.createBetMessage(bet)

	if err := cp.sendMessage(message); err != nil {
		return nil, fmt.Errorf("failed to send bet: %v", err)
	}

	responseServer, err := cp.receiveMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %v", err)
	}

	response, err := cp.parseResponseFromServer(responseServer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return response, nil
}

// createBetMessage serializes BetInfo struct in Client
// Messages are styled: "BET/<name>/<surname>/<document>/<birthdate>/<number>\n"
func (cp *ClientProtocol) createBetMessage(bet BetInfo) string {
	return fmt.Sprintf("BET/%s/%s/%s/%s/%s/%s\n",
			bet.Agency,
			bet.Name,
			bet.Surname,
			bet.Document, 
			bet.Birthdate,
			bet.Number)
}

// sendAll receives a byte array and loops until every byte is sent, avoiding short-writes
func (cp* ClientProtocol) sendAll(conn net.Conn, buffer []byte) error {
	totalSent := 0
	for totalSent < len(buffer) {
		sent, err := cp.conn.Write(buffer[totalSent:])
		if err != nil {
			return err
		}
		totalSent += sent
	}
	return nil
}

// sendMessage sends a string through the socket
func (cp *ClientProtocol) sendMessage(message string) error {
	if cp.conn == nil {
		return fmt.Errorf("socket closed")
	}

	msgLen := len(message)
	msgSize := uint16(msgLen)
	sizeBuff := make([]byte, 2)
	binary.BigEndian.PutUint16(sizeBuff, msgSize)

	if err := cp.sendAll(cp.conn, sizeBuff); err != nil {
		log.Errorf("action: send_message | result: fail | error: %v")
		return err
	}

	msgData := []byte(message)

	if err := cp.sendAll(cp.conn, msgData); err != nil {
		log.Errorf("action: send_message | result: fail | error: %v")
		return err
	}
	
	return nil
}

// receiveMessage receives server answer
// handles short-reads
func (cp *ClientProtocol) receiveMessage() (string, error) {
	if cp.conn == nil {
		return "", fmt.Errorf("socket closed")
	}

	reader := bufio.NewReader(cp.conn)
	var message []byte

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}

		if b == '\n' {
			return strings.TrimSpace(string(message)), nil
		}

		message = append(message, b)
	}
}

// parseResponseFromServer receives the message from the server and verifies its contents
func (cp* ClientProtocol) parseResponseFromServer(response string) (*ServerResponse, error) {
	parts := strings.Split(response, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid response format")
	}

	if parts[0] != "RESPONSE" {
		return nil, fmt.Errorf("unexpected response type: %s", parts[0])
	}

	return &ServerResponse {
		Type: parts[0],
		Status: parts[1],
		Message: parts[2],
	}, nil
}