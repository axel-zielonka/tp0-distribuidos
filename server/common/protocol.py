import logging

class Protocol:

    def __init__(self, sock):
        self._sock = sock

    # handles meesage receiving through the socket, avoiding short-reads
    def receive_message(self) -> str:
        message = b""
        while True:
            chunk = self._sock.recv(1024)
            if not chunk:
                break
            message += chunk
            if b"\n" in message:
                break
        return message.decode("utf-8").strip()

    # receives a string message and sends it through the socket, avoiding short writes
    def send_message(self, message: str):
        data = message.encode("utf-8")
        total_sent = 0
        while total_sent < len(data):
            sent = self._sock.send(data[total_sent:])
            if sent == 0:
                raise RuntimeError("Socket connection broken")
            total_sent += sent

    def close(self):
        try:
            self._sock.close()
        except Exception as e:
            logging.info("action: close_server_socket | result: success")
