import logging
from .utils import Bet

NO_WINNER = "NONE"
FINISH_MESSAGE = "F"

class Protocol:

    def __init__(self, sock):
        self._sock = sock

    # handles meesage receiving through the socket, avoiding short-reads
    def receive_message(self) -> str:
        msgSize = int.from_bytes(self._sock.recv(2), byteorder='big')        
        
        message = b""
        while len(message) < msgSize:
            chunk = self._sock.recv(msgSize - len(message))
            if not chunk:
                raise ConnectionError("Socket connection closed")
            message += chunk
        return message.decode("utf-8")

    # receives a string message and sends it through the socket, avoiding short writes
    def send_message(self, message: str):
        data = message.encode("utf-8")
        total_sent = 0
        while total_sent < len(data):
            sent = self._sock.send(data[total_sent:])
            if sent == 0:
                raise RuntimeError("Socket connection broken")
            total_sent += sent

    # receives a chunk of data through the socket and deserializes it to a list of Bets
    def parse_bets(self, bet_list, bets_read):
        msg = self.receive_message()

        if msg == FINISH_MESSAGE:
            return bet_list, True
        
        bets_str = msg.split(';')

        for bet in bets_str:
            bet_info = bet.split('/')
            if len(bet_info) != 6:
                logging.info(f"action: apuesta_recibida | result: fail | cantidad: {bets_read}")
                raise ValueError("Invalid bet")
            
            new_bet = Bet(bet_info[0], bet_info[1], bet_info[2], bet_info[3], bet_info[4], bet_info[5])

            bet_list.append(new_bet)
        return bet_list, False


    # reads the socket for the total amount of bets expected and then reads chunks until there are 
    # no more things to read. If bets_read == bet_count then it returns the Bet list, if not it returns None
    def receive_bets(self):
        bet_count = int.from_bytes(self._sock.recv(2), byteorder='big')
        bets_read = 0
        bets = []
        finished = False
        try:
            while not finished:
                bets, finished = self.parse_bets(bets, bet_count)
                bets_read = len(bets)
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")

        return bets, bets_read

    def get_string_result(self, winners):
        if len(winners) == 0:
            msg = NO_WINNER
        else:
            msg = ";".join([f"{winner.document}" for winner in winners])
        msg += '\n'
        return msg

    def close(self):
        try:
            self._sock.close()
        except Exception as e:
            logging.info("action: close_server_socket | result: success")
