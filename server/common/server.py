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
            message = protocol.receive_message()
            logging.debug(f"action: receive_message | result: success | ip: {addr[0]} | msg: {message}")
            response = self.__process_bet_message(message)

            protocol.send_message(response)
        except Exception as e:
            logging.error(f"action: handle_client | result: fail | error: {e}")
            try:
                error_response = "RESPONSE/ERROR/Error procesando apuesta\n"
                protocol.send_message(error_response)
            except:
                pass
        finally:
            protocol.close()

    def __process_bet_message(self, message):
        try:
            parts = message.split('/')
            if len(parts) != 7 or parts[0] != 'BET':
                return "RESPONSE/ERROR/Formato de mensaje invalido\n"
    
            _, agency, name, surname, document, birthdate, number = parts
            
            bet = Bet(agency, name, surname, document, birthdate, number)

            store_bets([bet])

            logging.info(f"action: apuesta_almacenada | result: success | dni: {document} | numero: {number}")
            return "RESPONSE/SUCCESS/Apuesta registrada correctamente\n"
        except Exception as e:
            logging.error(f"action: process_bet | result: fail | error: {e}")
            return "RESPONSE/ERROR/Error\n"


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
