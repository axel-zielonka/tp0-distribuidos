import socket
import logging
from common import utils
from .protocol import Protocol
import threading

BET_MESSAGE = "B"
FINISH_MESSAGE = "F"
ASK_FOR_RESULT_MESSAGE = "R"

class Server:
    def __init__(self, port, listen_backlog, clients):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True
        self.clients = clients
        self.already_finished_clients = 0
        self.winners = []
        self.lock = threading.Lock()
        self.threads = []

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        try:
            while self._running:
                try:
                    client_sock = self.__accept_new_connection()
                    if client_sock:
                        client_thread = threading.Thread(target=self.__handle_client_connection, args=(client_sock,))
                        client_thread.daemon = True
                        client_thread.start()
                        self.threads.append(client_thread)
                except Exception as e:
                    if self._running:
                        logging.error(f"action: handle_connection | result: fail | error: {e}")
                        continue  # Ignore unless shutdown is requested
                    else:
                        break
                self.__wait_for_threads()
        finally:
            self.__close_server_socket()
            logging.info("action: server_run | result: success")

    def __wait_for_threads(self):
        for thread in self.threads:
            thread.join()

    def shutdown(self):
        logging.info("action: server_shutdown | result: in_progress")
        self._running = False
        if self._server_socket:
            try:
                self._server_socket.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
        self.__wait_for_threads()
        logging.info("action: server_shutdown | result: success")

    def __close_server_socket(self):
        if self._server_socket:
            self._server_socket.close()
            logging.info("action: close_server_socket | result: success")
            self._server_socket = None
    
    def __handle_client_connection(self, client_sock):
        protocol = Protocol(client_sock)
        try:            
            message_type = protocol.receive_message()
            bet_count = 0
            response = ""
            if message_type == BET_MESSAGE:
                bets, bet_count = protocol.receive_bets()
                if bets == None:
                    response = f"ERROR/Unknown/{bet_count}\n"
                    logging.info(f"action: apuesta_recibida | result: fail | cantidad: {bet_count}")
                    protocol.send_message(response)
                else:
                    with self.lock:
                        self.already_finished_clients += 1
                    response = f"SUCCESS/SUCCESS/{bet_count}\n"
                    logging.info(f"action: apuesta_recibida | result: success | cantidad: {bet_count}")
                    protocol.send_message(response)
                    with self.lock:
                        utils.store_bets(bets)
            else:
                agency = protocol.receive_message()
                with self.lock:
                    if int(self.already_finished_clients) == int(self.clients):
                        if not self.winners:
                            bets = utils.load_bets()
                            winners = [bet for bet in bets if utils.has_won(bet)]
                            self.winners = winners
                            logging.info(f"action: sorteo | result: success")
                        
                        agency_winners = [bet for bet in self.winners if bet.agency == int(agency)]
                        response = protocol.get_string_result(agency_winners)
                        protocol.send_message(response)
        except Exception as e:
            logging.error(f"action: handle_client | result: fail | error: {e}")
            try:
                error_response = "ERROR/Unknown/0\n"
                protocol.send_message(error_response)
            except:
                pass
        finally:
            protocol.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
