import socket
import logging
from .utils import Bet, store_bets
from .protocol import Protocol


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True

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
                    addr = client_sock.getpeername()
                    protocol = Protocol(client_sock)
                    self.__handle_client_connection(protocol, addr)
                except Exception as e:
                    if self._running:
                        logging.error(f"action: handle_connection | result: fail | error: {e}")
                        continue  # Ignore unless shutdown is requested
                    else:
                        break
        finally:
            self.__close_server_socket()
            logging.info("action: server_run | result: success")

    def shutdown(self):
        logging.info("action: server_shutdown | result: in_progress")
        self._running = False
        if self._server_socket:
            try:
                self._server_socket.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass

    def __close_server_socket(self):
        if self._server_socket:
            self._server_socket.close()
            logging.info("action: close_server_socket | result: success")
            self._server_socket = None
    
    def __handle_client_connection(self, protocol: Protocol, addr):
        try:
            bets, bet_count = protocol.receive_bets()
            response = ""
            if bets == None:
                response = f"ERROR/Unknown/{bet_count}\n"
                logging.info(f"action: apuesta_recibida | result: fail | cantidad: {bet_count}")
            else:
                response = f"SUCCESS/SUCCESS/{bet_count}\n"
                store_bets(bets)
                logging.info(f"action: apuesta_recibida | result: success | cantidad: {bet_count}")

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
