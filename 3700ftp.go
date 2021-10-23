package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Server name we will be using for this project.
const Hostname = "ftp.3700.network"

func main() {
	// grab all command line inputs from program initialization.
	hostname, port, remotePath, localPath, localLocation, user, pass, command := parseInputs()
	// create a control connection to remote host implied from commandline arguments.
	conn, err := createConnection(hostname, port)
	if err != nil {
		panic("Error connecting to server: " + err.Error())
	}
	Read(conn)

	// login and setup connection for communication.
	conn = loginAndInitialize(conn, user, pass)

	// run operations instructed by user.
	handleCommands(conn, command, remotePath, localPath, localLocation)

	// Clean up connection and read final message.
	defer conn.Close()
	Read(conn)
	Write(conn, "QUIT\r\n")
	Read(conn)
}

// Parses all input params from the command line.
// RETURNS:
// 	- hostname - where we are connecting
// 	- port - port on host where we are connecting
//	- remotePath - path in remote host for which we are operating on
// 	- localPath - path on local host for which we are to possibly operate on
// 	- localLocation - index for which localPath came in relative to remotePath.
//		- 0, if before
//		- 1, if after
//		- -1, if not present
//	- username - users username for logging into remote host
//	- pass - users password for logging into remote host
// 	- command - command to be orchestrated in program
// THROWS IF: user provides an invalid command.
func parseInputs() (string, string, string, string, int, string, string, string) {
	args := os.Args[1:]
	if len(args) == 0 {
		panic("User must provide a valid command")
	}
	command := args[0]
	params := args[1:]
	isValidCommand(command, params)
	hostname, port, remotePath, localPath, localLocation, username, password := parseParams(params)
	return hostname, port, remotePath, localPath, localLocation, username, password, command
}

// Parses required params to orchestrate a connection to a remote host from an ftp remote url.
// RETURNS:
// 	- hostname - where we are connecting
// 	- port - port of host where we are connecting
//	- remotePath - path in remote host for which we are operating on
//	- username - users username for logging into remote host.
//	- pass - users password for logging into remote host.
func parseConnectionInfoFromUrl(url string) (string, string, string, string, string) {
	strippedUrl := url[7:]
	if !strings.Contains(strippedUrl, "/") {
		strippedUrl += "/"
	}
	splitParams := strings.Split(strippedUrl, "@")
	var hostname string
	port := "21"
	var remotePath string
	// case with no username and password as there was no "@" in param
	if len(splitParams) == 1 {
		panic("Url params must supply a username and password in the form: ftps://<username>:<password>@<hostname>/<path>")
	}
	userAndPass := strings.Split(splitParams[0], ":")
	if len(userAndPass) != 2 {
		panic("Urls must either be of form: ftps://<username>:<password>@<hostname>")
	} else {
		hostname = splitParams[1][:strings.IndexByte(splitParams[1], '/')]
		if strings.Contains(hostname, ":") {
			hostAndPort := strings.Split(hostname, ":")
			hostname = hostAndPort[0]
			port = hostAndPort[1]
		}
		remotePath = splitParams[1][strings.IndexByte(splitParams[1], '/'):]
		return hostname, port, remotePath, userAndPass[0], userAndPass[1]
	}
}

// Parses input param list for hostname, remotePath, username, password.
// RETURNS:
// 	- hostname - where we are connecting
// 	- port - port on host where we are connecting
//	- remotePath - path in remote host for which we are operating on
// 	- localPath - path on local host for which we are to possibly operate on
// 	- localLocation - index for which localPath came in relative to remotePath.
//		- 0, if before
//		- 1, if after
//		- -1, if not present
//	- username - users username for logging into remote host.
//	- pass - users password for logging into remote host.
func parseParams(params []string) (string, string, string, string, int, string, string) {
	if len(params) == 1 {
		hostname, port, remotePath, username, password := parseConnectionInfoFromUrl(params[0])
		return hostname, port, remotePath, "", -1, username, password
	} else if len(params) == 2 {
		if isValidURL(params[0]) {
			hostname, port, remotePath, username, password := parseConnectionInfoFromUrl(params[0])
			return hostname, port, remotePath, params[1], 1, username, password
		} else if isValidURL(params[1]) {
			hostname, port, remotePath, username, password := parseConnectionInfoFromUrl(params[1])
			return hostname, port, remotePath, params[0], 0, username, password
		} else {
			panic("No given params were a url!")
		}
	} else {
		panic("Invalid number of params!")
	}
}

// Throw error if input command and params don't map to an command that can be executed.
// RETURNS: nothing
// THROWS WHEN:
// - command exists but expects 1 param, but was given either too many or too little.
// - command exists but expects 2 params, but was given either too many or too little.
// - command doesn't exist.
func isValidCommand(command string, params []string) {
	if command == "ls" || command == "mkdir" || command == "rm" || command == "rmdir" {
		if len(params) > 1 || !isValidURL(params[0]) {
			panic(fmt.Sprintf("%s command must take the form: %s <url>", command, command))
		}
		// do nothing
	} else if command == "cp" || command == "mv" {
		if len(params) == 2 && isValidURL(params[0]) != isValidURL(params[1]) {
			// do nothing
		} else {
			panic(fmt.Sprintf("%s command must take the form: %s <url> <local> or %s <local> <url>", command, command, command))
		}
	} else {
		panic("Please input a valid command")
	}
}

// Is the given url a valid ftp url / does it begin with ftps:// ?
func isValidURL(url string) bool {
	return strings.HasPrefix(url, "ftps://")
}

// Creates a non-tls connection to a given host at a given port.
// RETURNS: the newly created non-tls connection, or an error if some issue occured.
func createConnection(hostname string, port string) (net.Conn, error) {
	return net.DialTimeout("tcp", fmt.Sprintf("%s:%s", hostname, port), 4 * time.Second)
}

// Writes the given message to given connection.
// Returns: byte array of response or an error (if one occured).
// THROWS IF: there was an issue writing to a given connection.
// REUSED FROM MY PROJ1
func Write(conn net.Conn, message string) string {
	num, err := conn.Write([]byte(message))
	if err != nil {
		panic("Error writing to server" + err.Error())
	}
	return fmt.Sprint(num)
}

// Reads incoming messages from a either tls or non-tls connection.
// RETURNS: A completed string message from currently connected server or an error if there was a connection issue.
// THROWS IF: there was an issue reading from the server.
// REUSED FROM MY PROJ1
func Read(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	lines, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return ""
		}
		panic("Error reading from server" + err.Error())
	}
	fmt.Println(lines)
	return lines
}

// Reads multi-line responses from tls or non-tls connection.
// RETURNS: nothing
// THROWS IF: there was an issue reading from the server.
func ReadAll(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		// while there are lines to read, read and print them
		lines, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			panic("Error reading from server")
		}
		fmt.Println(lines)
	}
}

// Wraps a connection with tls handshake.
// RETURNS: tls wrapped connection.
func authorize(conn net.Conn) net.Conn {
	return tls.Client(conn, &tls.Config{ServerName: Hostname})
}

// Initialize a secure connection to server by wrapping connection in TLS, logging in a user, and setting connection to private.
// Done by:
// - requesting connection use tls encryption via "AUTH TLS" command.
// - logging in user via "USER" and "PASS" commands.
// - setting connection to private via "PBSZ" and "PROT" commands.
// RETURNS: TLS wrapped control connection.
func loginAndInitialize(conn net.Conn, username string, password string) net.Conn {
	// TODO: find out how to handle anon sessions
	Write(conn, "AUTH TLS\r\n")
	Read(conn)
	conn = authorize(conn)
	Write(conn, fmt.Sprintf("USER %s\r\n", username))
	Read(conn)
	Write(conn, "PBSZ 0\r\n")
	Read(conn)
	Write(conn, "PROT P\r\n")
	Read(conn)
	Write(conn, fmt.Sprintf("PASS %s\r\n", password))
	Read(conn)
	return conn

}

// Orchestrates interaction with server based on input command.
// RETURNS: nothing
// THROWS IF: given command doesn't exist.
func handleCommands(controlConnection net.Conn, command string, remotePath string, localPath string, localLocation int) {
	switch command {
	case "mkdir", "rmdir", "rm":
		Write(controlConnection, translateCommand(command, remotePath, localLocation))
		break
	case ("ls"):
		// prepare a data channel which can transmit encrypted messages.
		dataConnection := handleDataSocket(controlConnection, translateCommand(command, remotePath, localLocation))
		// Read a possibly multi-line response from data channel.
		ReadAll(dataConnection)
		// close channel and read final message
		dataConnection.Close()
		break
	case "cp", "mv":
		// prepare a data channel which file data can be transmitted.
		setupDataTransfer(controlConnection)
		dataConnection := handleDataSocket(controlConnection, translateCommand(command, remotePath, localLocation))
		if (dataConnection == nil) {
			return
		}
		if command == "cp" {
			copy(controlConnection, dataConnection, localPath, remotePath, localLocation)
		} else {
			move(controlConnection, dataConnection, localPath, remotePath, localLocation)
		}
	default:
		panic("command does not exist")
	}
	return
}

// initialize a data connection between client and server, for which data (specifically file data and dir info) can be recieved and sent.
// RETURNS: two connection able to read from and write data to, same location, one is tls and one is not.
func handleDataSocket(controlConnection net.Conn, command string) (net.Conn) {
	// request info for creating data connection. i.e. ip and port to connect.
	Write(controlConnection, "PASV\r\n")
	res := Read(controlConnection)
	ip, port := readPortAndIP(res)
	// write command which will utilize data connection.
	Write(controlConnection, command)
	// create connection from information given from PASV response
	dataConnection, err := createConnection(ip, port)
	if err != nil {
		panic("Error connecting data socket to server: " + err.Error())
	}
	// read control response from command that will utilize data connection.
	// If control response has error status code, then abort
	res = Read(controlConnection)
	resCode := res[0]
	if (checkErrorResponse(rune(resCode))) {
		dataConnection.Close()
		return nil;
	}
	// wrap data connection in a tls handshake.
	dataConnection = authorize(dataConnection)
	// return the tls wrapped data connection.
	return dataConnection
}

// Check if given char is equal to the first number in an error code
// RETURNS: if first char is equal to the first number in an error code
func checkErrorResponse(resHeader rune) bool {
	return (resHeader == '5' || resHeader == '4' || resHeader == '6')

}

// Tell server to prepare for data transfer over some data connection.
// Done by:
//    - Setting server to 8-bit data mode ("TYPE I")
//    - Setting server to stream mode ("MODE S")
//    - Setting server to file-oriented mode ("STRU F")
// RETURNS: this function is void.
func setupDataTransfer(controlConnection net.Conn) {
	Write(controlConnection, "TYPE I\r\n")
	Read(controlConnection)
	Write(controlConnection, "MODE S\r\n")
	Read(controlConnection)
	Write(controlConnection, "STRU F\r\n")
	Read(controlConnection)
}

// Parse an ip and port from a given message and throw an error if message is malformed (if you couldn't find said ip and port)
// RETURNS: string of four numbers seperated by "." representing an ip and a single number string representing port.
// returned port must be able to be represented by 16 bits.
func readPortAndIP(message string) (string, string) {
	splitMessage := strings.Split(message, " ")
	if len(splitMessage) == 5 && splitMessage[0] == "227" {
		ipAndPort := strings.Split(splitMessage[4][1:len(splitMessage[4])-4], ",")
		ip := ipAndPort[:4]
		firstBitsOfPort, err1 := strconv.Atoi(ipAndPort[4])
		lastBitsOfPort, err2 := strconv.Atoi(ipAndPort[5])
		if err1 != nil || err2 != nil {
			panic("Invalid port values from server were given")
		}
		port := (firstBitsOfPort << 8) + lastBitsOfPort
		return strings.Join(ip, "."), fmt.Sprint(port)

	} else {
		panic("Invalid response from server.")
	}

}

// Translate command line commands into commands that can be excecuted on CS3700's server.
// RETURNS: a valid command for CS3700's server or an error if command is invalid.
func translateCommand(command string, remotePath string, localLocation int) string {
	// TODO: cp may be store or retr based on @localLocation
	pathAndCloser := remotePath + "\r\n"
	switch command {
	case ("ls"):
		return "LIST " + pathAndCloser
	case ("mkdir"):
		return "MKD " + pathAndCloser
	case ("rm"):
		return "DELE " + pathAndCloser
	case ("rmdir"):
		return "RMD " + pathAndCloser
	case "cp", "mv":
		if localLocation == 1 {
			return "RETR " + pathAndCloser
		} else if localLocation == 0 {
			return "STOR " + pathAndCloser
		} else {
			panic("cp and mv commands require a local file path.")
		}
	default:
		panic("command does not exist")
	}
}

// Moves a file from one host to another (file no longer exists at intial location after process is done)
// RETURNS: this function is void.
func move(controlConnection net.Conn, dataConnection net.Conn, localPath string, remotePath string, localLocation int) {
	copy(controlConnection, dataConnection, localPath, remotePath, localLocation)
	res := Read(controlConnection)
	if (checkErrorResponse(rune(res[0]))) {
		return;
	}
	if localLocation == 0 {
		err := os.Remove(localPath)
		if err != nil {
			fmt.Println("Error deleting file: " + err.Error())
		}

	} else if localLocation == 1 {
		Write(controlConnection, fmt.Sprintf("DELE %s\r\n", remotePath))
		Read(controlConnection)
		// handleCommands(controlConnection, "rm", remotePath, "", -1, "")

	} else {
		panic("invalid inputs to move")
	}
}

// Copies a file from one host to another (after process an identical file should be present at local and remote paths).
// RETURNS: this function is void.
func copy(controlConnection net.Conn, dataConnection net.Conn, localPath string, remotePath string, localLocation int) {
	defer dataConnection.Close()
	if localLocation == 0 {
		file, err := os.Open(localPath) // For read access.
		if err != nil {
			fmt.Println("Error reading from local file: " + err.Error())
			return
		}
		defer file.Close()
		_, err = io.Copy(dataConnection, file)
		if err != nil {
			fmt.Println("Error reading from local file: " + err.Error())
			return
		}
	} else if localLocation == 1 {
		splitPath := strings.Split(localPath, "/")
		fileName := splitPath[len(splitPath)-1]
		file, err := os.Create(fileName)
		if err != nil {
			fmt.Println("Error creating local file: " + err.Error())
			return
		}
		defer file.Close()
		_, err = io.Copy(file, dataConnection)
		if err != nil {
			fmt.Println("Error writing from local file: " + err.Error())
			return
		}
	} else {
		panic("Invalid inputs")
	}
}
