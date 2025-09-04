# TP0: Docker + Comunicaciones + Concurrencia

## Instrucciones de uso
El repositorio cuenta con un **Makefile** que incluye distintos comandos en forma de targets. Los targets se ejecutan mediante la invocación de:  **make \<target\>**. Los target imprescindibles para iniciar y detener el sistema son **docker-compose-up** y **docker-compose-down**, siendo los restantes targets de utilidad para el proceso de depuración.

Los targets disponibles son:

| target  | accion  |
|---|---|
|  `docker-compose-up`  | Inicializa el ambiente de desarrollo. Construye las imágenes del cliente y el servidor, inicializa los recursos a utilizar (volúmenes, redes, etc) e inicia los propios containers. |
| `docker-compose-down`  | Ejecuta `docker-compose stop` para detener los containers asociados al compose y luego  `docker-compose down` para destruir todos los recursos asociados al proyecto que fueron inicializados. Se recomienda ejecutar este comando al finalizar cada ejecución para evitar que el disco de la máquina host se llene de versiones de desarrollo y recursos sin liberar. |
|  `docker-compose-logs` | Permite ver los logs actuales del proyecto. Acompañar con `grep` para lograr ver mensajes de una aplicación específica dentro del compose. |
| `docker-image`  | Construye las imágenes a ser utilizadas tanto en el servidor como en el cliente. Este target es utilizado por **docker-compose-up**, por lo cual se lo puede utilizar para probar nuevos cambios en las imágenes antes de arrancar el proyecto. |
| `build` | Compila la aplicación cliente para ejecución en el _host_ en lugar de en Docker. De este modo la compilación es mucho más veloz, pero requiere contar con todo el entorno de Golang y Python instalados en la máquina _host_. |


## Parte 1: Introducción a Docker
En esta primera parte del trabajo práctico se plantean una serie de ejercicios que sirven para introducir las herramientas básicas de Docker que se utilizarán a lo largo de la materia. El entendimiento de las mismas será crucial para el desarrollo de los próximos TPs.

### Ejercicio N°1:

Se generó un script directamente en bash. Este script primero hace un chequeo de la cantidad de variables recibidas, en caso de que no sea 2 devuelve un código de error 1. Luego hace un chequeo para ver si la cantidad de clientes ingresados es un número entero mayor a 0, en caso de que no lo sea devuelve código de error 2.

![Código de error](img/ej1_img1.png)

Una vez hecho el chequeo de erorres, se comienza a reescribir el archivo de salida, cuyo path esta guardado en la variable `$OUTPUT_FILE`. Con el comando `cat > "$OUTPUT_FILE` se redirige la salida estándar al archivo de salida, reescribiendo su contenido. `<< 'END'` inicia un `here-doc`, que indica que se escriba todo lo que se encuentra a continuación hasta que se encuentre una línea que contenga únicamente el delimitador `END`. 

![Escribir el server](img/ej1_img2.png)

Una vez escrito el server, se pasan a escribir los clientes. El script realiza un for desde 1 hasta la cantidad de clientes. En cada iteración se escribe un cliente. En este caso se utiliza `cat >> "$OUTPUT_FILE` ya que `>>` indica que se agregue al archivo en vez de sobreescribir.

![Escribir clientes](img/ej1_img3.png)

Por último se vuelve a escribir la red sin modificaciones a la original. 

![Escribir network](img/ej1_img4.png)

El resultado final luego de ejecutar para 5 clientes es:

![Resultado final](img/ej1_img5.png)

El script se invoca de la siguiente forma:

> `./generar-docker.sh <archivo_de_salida.yml> <num_clients>`


### Ejercicio N°2:

Se agregaron los volumenes para el servidor y los clientes en el script del `docker-compose-dev.yaml` y se eliminaron las variables de entorno del nivel de logging en ambos casos ya que son parte de la configuración y no deberían estar "hardcodeadas" en el archivo. 

![Cambios en el código](img/ej2_img1.png)

### Ejercicio N°3:

El script `validar-echo-server.sh` busca en el archivo `docker-compose-dev.yaml` el nombre del proyecto. El comando `"$(awk '/^[[:space:]]*name:[[:space:]]*/{print $2; exit}' "$COMPOSE_FILE" 2>/dev/null || echo tp0)"` busca en el compose la primera aparición de `name` y asigna su valor a la variable `PROJECT_NAME`. Cabe aclarar que en caso de no encontrar el nombre, se elige por defecto `tp0` ya que era el nombre original. Luego de asignar el nombre del proyecto, asigna el nombre de la red en la variable `NETWORK`

![Búsqueda de nombre y asignación de la red](img/ej3_img1.png)

Una vez hecho esto, se busca entre los containers en ejecución al `server`, ya que este es el que maneja la conexión y es a donde se va a enviar el mensaje. En caso de no encontrarlo, el script devuelve `fail` automáticamente sin posibilidad de reconexión.

![Buscar 'server' en los containers](img/ej3_img2.png)

Si se encontró el container del servidor, se procede a encontrar el puerto por donde enviar la información. Para que no sea un valor hardcodeado, se ejecuta dentro de docker una terminal de python y se lee el archivo `config.ini` del servidor para obtener el puerto. Si no se encontrara un puerto, se asigna el puerto default del archivo, y en caso de que tampoco haya uno se asigna por defecto el puerto `12345`. Aclaración, se usa el puerto `12345` como default ya que en el archivo original del `config.ini` ese era el valor. 

Luego se hace una validación para ver si el puerto leído tiene un valor correcto, es decir, que sea un número.

![Búsqueda y validación del puerto](img/ej3_img3.png)

Una vez obtenido el puerto, se levanta un contenedor temporal de docker con `docker run -rm` y se conecta a `NETWORK` usando alpine, que ejecuta una shell. Envia por `stdin` a `netcat` conectado con `server` en el puerto `PORT` el mensaje inicial sin el salto de línea. 

Por último, chequea que el mensaje original sea igual al mensaje recibido devolviendo según corresponda. 

![Conexión y resultado](img/ej3_img4.png)

Modo de uso del script:

> `./validar-echo-server.sh`

### Ejercicio N°4:

Resolución en `client`:
En `client/common/client.go` se implementaron las siguientes funciones:
* `closeConnection()`: realiza el cierre del socket, liberando el file descriptor y terminando la comunicación con el server. Esta función fue creada para no estar usando `c.conn.Close()` y mejorar la legibilidad del código
![Función closeConnection](img/ej4_img1.png)

* En `StartClientLoop()` se modificaron algunas cosas. Primero, ahora la función recibe por parámetro un `context.Context` para poder manejar el shutdown. Además en el loop, se chequea si la conexión fue terminada con `ctx.Done()`, y en el caso afirmativo cierra la conexión y termina el loop. 
* Se modificó el sleep entre mensajes, ya que `time.Sleep` bloquea la goroutine y no puede interrumpirse hasta que no pase el tiempo. Se agregó una condición de `select` para que si se interrumpió con un `SIGTERM` durante ese momento no haya que esperar a que termine el sleep para qeu termine la conexión. El sleep ahora se maneja con un `time.After` que sí puede ser interrumpido por una señal de shutdown

![StartClientLoop](img/ej4_img2.png)
![StartClientLoop2](img/ej4_img3.png)

En `client/main.go`:
* Se crea un `context` para poder mandar información entre goroutines, junto con su función `cancel`.
* Se crea un canal buffereado `sigChan` que recibe señales del sistema operativo. Como en este caso solamente se pide que se maneje la señal `SIGTERM` el tamaño del buffer es de 1. El canal queda a la espera de señales.
* Se lanza una gorotutine para manejar las señales, con una variable `sig` que se bloquea hasta recibir algo desde `sigChan`. Una vez que recibe, se loguea el shutdown y se llama a la función `cancel()` del `context`

![main.go](img/ej4_img4.png)

Resolución en `server`:
En `server/common/server.py`:
* Se implementó un flag `_running` que indica si el server está activo

![Flag running](img/ej4_img5.png)

* En `run()` se modificó el loop de iteraciones. Antes era un `while True` y ahora pasa a ser un `while self._running`, que hace que al momento en el que el flag deja de ser `True`, el loop se corte (terminando la iteración actual). La aceptación de nuevas conexiones y el manejo de las conexiones de los clientes está dentro de un bloque `try` ya que si sucede un error quiero poder manejarlo correctamente y que no se corte el programa. Una vez que termina el `while`, se hace un cierre del socket del servidor llamando a la función `__close_server_socket()`.

![Run](img/ej4_img6.png)

* Se implementó `shutdown()`, que cambia el valor del flag a `False` y hace un `socket.shutdown(RDWR)`
* Se implementó `__close_server_socket()` que hace un `socket.close()` y libera los recursos.

![Shutdown y close](img/ej4_img7.png)

En `server/main.py`:
* Se implementó `signal_handler()` que recibe el número de señal, el frame (estado actual de la pila de ejecución) y el `server`. Cabe aclarar que si bien `frame` no se utiliza en la función, es necesario que esté en la firma de la función para el handling de señales en python. En esta función se hace el log del shutdown y se llama a `server.shutdown()`. 

![Signal Handler](img/ej4_img8.png)

* En `main()` se hace el llamado a `signal_handler()` a través de `signal.signal(signal.SIGTERM, lambda signum, frame: signal_handler(signum, frame, server))`. Se usa `lambda` porque `signal.signal` originalmente recibe 2 parámetros, y `lambda` me permite hacer el llamado a `signal_handler()`. Por último, se modificó la línea dónde estaba `server.run()` y ahora está dentro de un bloque `try`, ya que en caso de que falle se debería llamar a `server.shutdown()` y salir con código de error.

![Main](img/ej4_img9.png)

## Parte 2: Repaso de Comunicaciones

### Ejercicio N°5:

En el cliente, las apuestas se serializan de la siguiente forma `"BET/<Agencia>/<Nombre>/<Apellido>/<Documento>/<Fecha_de_nacimiento>/<Numero>\n"`. El caracter `\n` delimita el fin del mensaje y es lo que busca el servidor para cortar las lecturas.
* El envio de mensajes se encuentra en el metodo `sendMessage()`. En este, se crea un buffer de bytes con el texto que se quiere enviar junto con un contador de bytes enviados. Primero se envia por el socket el largo del mensaje, en 2 bytes en formato `big endian`. Luego, se entra en un loop en el que se van enviando bytes y actualizando el valor del contador. El loop puede terminar antes de tiempo por un error o puede finalizar correctamente cuando todos los bytes hayan sido enviados, evitando `short writes`

![Send](img/ej5_img1.png)

![Send](img/ej5_img2.png)

* La recepcion de los mensajes hace algo similar al envio, en la funcion `receiveMessage()` solamente que lee de a 1 byte a la vez. Esto se debe a que en el envio, el protocolo conoce la longitud del mensaje que va a enviar, pero en la recepcion no. Es por esto que hace una lectura de a 1 byte hasta recibir el caracter `\n` que indica el fin del mensaje, evitando de esta forma `short reads`

![Receive](img/ej5_img3.png)

Por el otro lado, en el servidor, los mensajes de respuesta se serializan de la siguiente forma `"RESPONSE/<success>/<message>\n"` donde `<success>` puede tomar valor `SUCCESS` o `ERROR` dependiendo de si el mensaje recibido era valido.
* El envio de mensajes se encuentra en el metodo `send_message()` y se comporta de forma muy similar al del envio del cliente. La diferencia es que no se envía el tamaño del mensaje ya que el cliente separa los mensajes por el `\n`, y como los mensajes que envía el servidor no tienen saltos de línea intermedios se puede asegurar que habra un único `\n` por mensaje, por lo que no es necesario enviar el tamaño. Se crea un array de bytes y se tiene un contador con los bytes enviados, para luego en loop enviar y actualizar el contador hasta que se termina correctamente y se envia todo u ocurre un error y se termina la comunicacion.

![Send](img/ej5_img4.png)

* La recepción de mensajes se encuentra en el método `receive_message()`. Primero lee del socket 2 bytes en formato `big endian` que indican el tamaño del mensaje que se va a recibir. Luego de esto, entra en un loop en el que intenta leer la cantidad de bytes que le faltan para llegar a ese tamaño que se leyó, evitando así `short-reads`

![Receive](img/ej5_img5.png)

### Ejercicio N°6:

> [!NOTE]
> Sobre el protocolo
> El protocolo fue modificado respecto al ejercicio 5 para adatparlo mejor a lo que pedía el ejercicio

Se modificó el script `generar-compose.sh` para inyectar la persistencia del archivo de apuestas. Además, se quitaron las variables de entorno ya que las apuestas ahora se leen del archivo `.csv` de cada agencia. 

![script](img/ej6_img1.png)

En el cliente, se modificaron las siguientes cosas:
* La lectura del archivo de cada agencia y la carga de las correspondientes apuestas se hace en `loadBetsFromFile` en el archivo `client/common/bet.go`. En esta función, se lee el archivo línea por línea y se cargan los datos. 

![loadBets](img/ej6_img2.png)

* En el archivo `client/common/client.go` se agregó la variable de configuración para el tamaño máximo del batch, que se lee desde el archivo de configuración. Además, en el inicializador del cliente se llama a la lectura del archivo de las apuestas.

* Los cambios más importantes se hicieron en el protocolo. Para comenzar, al ya no ser un único mensaje enviado al servidor, sino que son múltiples _batches_ de datos, fue necesario cambiar el formato de los envíos. En primer lugar, las apuestas pasaron de serializarse de `BET/<Agencia>/<Nombre>/<Apellido>/<Documento>/<Fecha de nacimiento>/<Numero>\n` a serializarse de la forma `<Agencia>/<Nombre>/<Apellido>/<Documento>/<Fecha de nacimiento>/<Numero>`. Lo siguiente que se hizo es que al agrupar varias apuestas en un mismo _batch_ éstas se separaban con un `;`, quedando entonces de la forma `Apuesta1;Apuesta2;Apuesta3;...;ApuestaN`. De esta forma, se reduce la cantidad de bytes por envío, haciendo más eficiente cada transmisión, ya que no se está enviando el tipo de mensaje `BET` antes de cada apuesta. Por otra parte, se modificaron también los métodos de envío de datos. En lugar de `sendBet()`, ahora se cuenta con `sendBets()` que primero envía el total de apuestas que se van a transmitir, y luego hace un loop para cada _batch_ llamando a `sendBatch()` que serializa cada apuesta y las une en una misma tira de bytes para el envío. Cabe aclarar que `sendBatch()` vendría a reemplazar a `sendBet()`, pero aplicado a múltiples apuestas, ya que envía el largo del mensaje y luego transmite los bytes haciendo uso de `sendAll()` que se asegura que no sucedan _short-writes_. Se hicieron modificaciones también en `receiveMessage()`. Se modificó la forma en la que se leía del socket. El uso de `bufio.NewReader` junto con `ReadByte` podían hacer que el socket quede bloqueado aún cuando se hacía un _close_ del mismo. Es por esto que se modificó y ahora se hace la misma lectura de un byte a la vez pero usando un buffer de tamaño 1, con `Conn.Read`, que si se cierra el socket se desbloquea. Por último desde el lado del protocolo del cliente, se agregó un campo `betCount` al struct `ServerResponse` reemplazando al campo `type`, para chequear que el número de apuestas que recibió el servidor sea el mismo que el número de apuestas enviadas por el cliente.

![sendBets](img/ej6_img3.png)

![sendBatch](img/ej6_img4.png)

![receiveMessage](img/ej6_img5.png)

![serverResponse](img/ej6_img6.png)

En el servidor se realizaron los siguientes cambios:
* Se eliminó la función `__process_bet_message` en `server/common/server.py` ya que la lógica no debería estar ahí sino en el protocolo.

* Al igual que en el cliente, se realizaron los cambios más significativos en el protocolo. Se implementó la función `receive_bets()` que lee del socket la cantidad de apuestas que se van a recibir. Esto se hace para tener como validación luego de leer todos los _batches_, si el número final de apuestas recibidas no coincide con el número de apuestas recibido originalmente significa que se produjo algún error en el envío o en la lectura. En esta función se entra en un loop que se repite mientras las apuestas leídas no sean las apuestas esperadas. En este loop, se llama a la función `parse_bets()`. Esta función llama a `receive_message()` y, separando por el caracter `;`, obtiene una lista de strings. Luego, para cada string recibido lo deserializa separando los componentes con el separador `/`. Luego almacena la apuesta recibida en caso de que tenga un formato válido, y si no devuelve un `ValueError`. Por último, se modificaron los mensajes enviados por el servidor. Antes tenían la forma `<TYPE>/<success>/<message>\n` y ahora pasan a tener la forma `<success>/<success>/<bets>\n`, donde el primer success indica si se procesaron con éxito las apuestas, el segundo repite en caso de éxito y en caso de error indica el tipo, y `bets` indica la cantidad de apuestas recibidas.

![receive_bets](img/ej6_img7.png)

![parse_bets](img/ej6_img8.png)

![handle_client](img/ej6_img9.png)


### Ejercicio N°7:

Se implementó en el protocolo del cliente la función `sendMessageType()`, que envía al servidor en 1 byte el tipo de mensaje que va a enviar después: B de Bet (apuesta), F de Finished o R de Results. 

![sendType](img/ej7_img1.png)

También se implementó la función `askForResults()`, que envía un mensaje de tipo R y se queda esperando a la respuesta del servidor. Esta función devuelve la cantidad de ganadores recibidos, o error en caso de que haya ocurrido un problema

![askForResults](img/ej7_img2.png)

En el loop del cliente, se agregó la lógica para la espera de los resultados. El cliente pide los resultados al servidor, y si no recibe respuesta se desconecta y hace un sleep. Pasado ese tiempo, se vuelve a conectar e intenta de vuelta. De esta forma, ningún cliente se queda bloqueado esperando a los resultados.

![loop](img/ej7_img3.png)

En el servidor se agregaron los siguientes atributos:
* `clients`: indica la cantidad de clientes que se esperan, se leen de una variable de entorno en el `docker-compose-dev.yaml`
* `already_finished_clients`: indica la cantidad de clientes que pidieron los resultados
* `winners`: es un listado con los ganadores

![atributos](img/ej7_img4.png)

Se agregó la lógica según el mensaje recibido. . En caso de que sea un mensaje `B`, se mantiene la lógica anterior. En caso de recibir un mensaje `F`, se corta el loop de recepción de apuestas que se encontraba en `receive_bets()` del protocolo. En caso de recibir un pedido de resultados, se hace el sorteo en caso de que no se haya hecho y luego se recibe la agencia correspondiente y se envían todas los ganadores de apuestas con el `id` de agencia correspondiente. 

![handleClient](img/ej7_img5.png)

![parseBets](img/ej7_img6.png)

![string](img/ej7_img7.png)

## Parte 3: Repaso de Concurrencia

### Ejercicio N°8:
Para resolver el problema del paralelismo, recurrí a utilizar _threads_ con _locks_ para los accesos a los recursos compartidos. Estos recursos son: el contador de clientes finalizados, el sorteo y el acceso a los archivos de apuestas y ganadores. Si bien es verdad que el Global Interpreter Lock no permite que más de un thread ejecute código de Python a la vez, ocurre que cuando un thread llama a una operación bloqueante, como puede ser un `recv` o un `send` de un socket, se libera el GIL para que lo pueda tomar otro thread. Y siendo esta una aplicación con muchas operaciones de I/O y no tan CPU-intensive, los threads se bloquean y le dan el espacio a otro rápidamente sin que esté mucho tiempo bloqueado. 

![img1](img/ej8_img1.png)

![im2](img/ej8_img2.png)

![img3](img/ej8_img3.png)

![img4](img/ej8_img4.png)

### Pruebas automáticas 
Se provee una captura de pantalla mostrando que el código pasa todas las pruebas provistas por la cátedra