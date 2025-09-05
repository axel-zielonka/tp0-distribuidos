package common

import (
	"fmt"
	"net"
	"strings"
	"encoding/binary"
	"strconv"
	"encoding/csv"
	"os"
	"io"

	"github.com/op/go-logging"
)

const MAX_BATCH_SIZE = 8192
const BET_MESSAGE = "B"
const FINISH_MESSAGE = "F"
const ASK_FOR_RESULT_MESSAGE = "R"
const NO_WINNERS = "NONE"


var protocolLog = logging.MustGetLogger("protocol")

type ClientProtocol struct {
	conn net.Conn
	id string
}

type ServerResponse struct {
	Status  string
	Message string
	betCount int
}


func NewClientProtocol(id string, conn net.Conn) (ClientProtocol) {
	return ClientProtocol{conn: conn, id: id}
}

// createBetMessage serializes BetInfo struct in Client
// Messages are styled: "BET/<name>/<surname>/<document>/<birthdate>/<number>\n"
func (cp *ClientProtocol) createBetMessage(bet BetInfo) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s",
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
	var message []byte
	buf := make([]byte, 1)
	for {
		b, err := cp.conn.Read(buf)
		if err != nil {
			return "", err
		}
		if b > 0 {
			b := buf[0]
			if b == '\n'{
				return strings.TrimSpace(string(message)), nil
			}
			message = append(message, b)
		}
	}
}

func parseLine(line []string, id string) BetInfo {
	return BetInfo{
		Agency:    id,
		Name:      line[0],
		Surname:   line[1],
		Document:  line[2],
		Birthdate: line[3],
		Number:    line[4],
	}
}

func readBet(reader *Reader, id string) (BetInfo, error) {
	line, err := reader.Read()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		bet := parseLine(line, id)
		return bet, nil
}

// sendBets first sends the total amount of bets, and then sends the bets in totalBets/maxBatchSize chunks.
func (cp* ClientProtocol) sendBets(conn net.Conn, maxBatchSize int) (error, int) {
	f, err := os.Open("agency.csv")
	if err != nil {
		log.Errorf("action: read_bet_file | result: fail | error: %v", err)
		return err, -1
	}
	defer f.Close()
	reader := csv.NewReader(f)
	batch := make([]BetInfo, 0, maxBatchSize)
	betCount := 0
	for {
		bet, err := readBet(reader, cp.id)
		if err != nil {
			return err, -1
		}
		batch = append(batch, bet)
		if len(batch) == maxBatchSize {
			err := cp.sendBatch(cp.conn, batch, betCount)
			if err != nil {
				return err, -1
			}
			batch = batch[:0] 
		}
		betCount += 1
	}
	if len(batch) > 0 {
		err := cp.sendBatch(cp.conn, batch, betCount)
		if err != nil {
			return err, -1
		}
	}
	log.Infof("action: send_bets | result: success")
	log.Infof("action: finishing_sending_bets | result: in_progress")
	err = cp.sendMessageType(cp.conn, FINISH_MESSAGE)
	if err != nil {
		log.Infof("action: finishing_sending_bets | result: fail | error: %v", err)
		return err, -1
	} else {
		log.Infof("action: finishing_sending_bets | result: success")
	}
	return nil, betCount
}

// sendBatch serializes a chunk of bets and sends it to the server, ensuring no short-writes
func (cp *ClientProtocol) sendBatch(conn net.Conn, batch []BetInfo, betsSent int) error {
	betsString := make([]string, 0, len(batch))

	for _, bet := range batch {
		betString := cp.createBetMessage(bet)

		betsString = append(betsString, betString)
	}

	batchString := strings.Join(betsString, ";")

	if len(batchString) > MAX_BATCH_SIZE {
		return fmt.Errorf("Batch size cannot exceed 8kb")
	}

	if err := cp.sendMessage(batchString); err != nil {
		return err
	}

	return nil
}

func (cp* ClientProtocol) sendMessageType(conn net.Conn, messageType string) error {
	return cp.sendMessage(messageType)
}

func (cp* ClientProtocol) changeSocket(conn net.Conn) {
	cp.conn = conn
}

func (cp* ClientProtocol) askForResults(conn net.Conn, messageType string) (int, error) {	
	if err := cp.sendMessageType(cp.conn, messageType); err != nil {
		return -1, err
	}
	agency := cp.id
	if err := cp.sendMessage(agency); err != nil {
		return -1, err
	}
	msg, err := cp.receiveMessage()
	if err != nil {
		return -1, err
	}
	if msg == NO_WINNERS {
		return 0, nil
	}
	winners := strings.Split(msg, ";")
	return len(winners), nil
}


// parseResponseFromServer receives the message from the server and verifies its contents
func (cp* ClientProtocol) parseResponseFromServer(response string) (*ServerResponse, error) {
	parts := strings.Split(response, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid response format")
	}

	bets, _ := strconv.Atoi(parts[2])

	return &ServerResponse {
		Status: parts[0],
		Message: parts[1],
		betCount: bets,
	}, nil
}