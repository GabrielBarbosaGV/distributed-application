package main

import (
	"fmt"
	"sync"
	"net"
	"bufio"
	"encoding/json"
)

// Server configuration strings
const hostName,
	primaryServerPort,
	requiringServerPort,
	network string =
		"localhost",
		"8000",
		"8001",
		"tcp"

const tcpAddrSolveErr,
	 tcpListenErr,
	 acceptTCPErr,
	 connReadErr,
	 unmarshalErr,
	 marshalErr,
	 primaryDialErr string =
		"solving the TCPAddr",
		"trying to listen to TCP packets",
		"trying to accept a TCP connection",
		"trying to read string from connection",
		"trying to unmarshal JSON",
		"try to marshal struct",
		"trying to dial the Primary Server"

const cldntDial,
	cldntReq string =
		"Internal Error: Could not dial Primary Server",
		"Internal Error: Could not generate proper request to Primary Server (Marshal Error)"

type serviceRequest struct {
	ServiceName string `json:"sn"`
	Values string `json:"vs"`
	EndConn bool `json:"ENDCONN"`
}

type serviceResponse struct {
	Values string `json:"vs"`
	EndConn bool `json:"ENDCONN"`
}

// Gera uma função para imprimir erros genericamente
func newGenericErrMsgr(serverType, network, address string) func (string, error) string {
	return func (errInstance string, err error) string {
		return fmt.Sprintf("There was an error in %s for the %s with network %s and address %s\nError was: %s", errInstance, serverType, network, address, err.Error())
	}
}

// Nesta aplicação, as funções primaryServer, requiringServer e client
// serão acionadas como goroutines independentes

// Servidor primário, oferece serviços ao cliente e ao servidor requerente
func primaryServer(network, address string) {
	serverType := "Primary Server"
	genericErrMsg := newGenericErrMsgr(serverType, network, address)

	// Os serviços oferecidos são funções guardadas em um map
	services := map[string] func(string) string{
		"rot13": func(tocipher string) string {
			bytesToCipher := []byte(tocipher)

			for i, v := range bytesToCipher {
				if v >= 65 && v <= 90 {
					bytesToCipher[i] = (((v - 65) + 13) % 26) + 65
				} else if v >= 97 && v <= 122 {
					bytesToCipher[i] = (((v - 97) + 13) % 26) + 97
				}
			}

			return string(bytesToCipher[:])
		},
		"unrot13": func(toUncipher string) string {
			bytesToUncipher := []byte(toUncipher)

			for i, v := range bytesToUncipher {
				if v >= 65 && v <= 90 {
					bytesToUncipher[i] = (((v - 65 + 26) - 13) % 26) + 65
				} else if v >= 97 && v <= 122 {
					bytesToUncipher[i] = (((v - 97 + 26) - 13) % 26) + 97
				}
			}

			return string(bytesToUncipher[:])
		},
	}

	tcpAddr, err := net.ResolveTCPAddr(network, address)

	if err != nil {
		fmt.Println(genericErrMsg(tcpAddrSolveErr, err))
		return
	}

	ln, err := net.ListenTCP(network, tcpAddr)
	
	if err != nil {
		fmt.Println(genericErrMsg(tcpListenErr, err))
		return
	}

	for {
		conn, err := ln.AcceptTCP()

		if err != nil {
			fmt.Println(genericErrMsg(acceptTCPErr, err))
		}

		go func (connection *net.TCPConn) {
			reader := bufio.NewReader(connection)

			for {
				message, err := reader.ReadString('\n')

				if err != nil {
					fmt.Println(genericErrMsg(connReadErr, err))
					connection.Close()
					// TODO Close in other error cases
					return
				}

				var req serviceRequest
				err = json.Unmarshal([]byte(message), &req)

				if err != nil {
					fmt.Println(genericErrMsg(unmarshalErr, err))
					return
				}

				if req.EndConn {
					fmt.Println("Ending this connection.")
					connection.Close()
					return
				}

				response := serviceResponse{ services[req.ServiceName](req.Values), true }

				res, err := json.Marshal(response)

				if err != nil {
					fmt.Println(genericErrMsg(marshalErr, err))
				}

				_, err := connection.Write(res)

				if err != nil {

				}
			}
		}(conn)
	}
}

// Servidor requerente, serve ao cliente mas requer serviços do primário
func requiringServer(network, address string) {
	serverType := "Requiring Server"
	genericErrMsg := newGenericErrMsgr(serverType, network, address)

	storedString := "DEFAULT"

	services := map[string] func (string) string {
		"store": func (toStore string) string {
			conn, err := net.DialTCP(network, nil, net.JoinHostPort(hostName, primaryServerPort))

			if err != nil {
				fmt.Println(genericErrMsg(primaryDialErr, err))
				return cldntDial
			}

			request := serviceRequest{ "rot13", storedString, false }

			req, err := json.Marshal(request)

			if err != nil {
				fmt.Println(genericErrMsg(marshalErr, err))
				return cldntReq
			}



			// TODO
			return "placeholder"
		}
	}
}

// Cliente, faz requisições
func client(wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
}

func main() {
	primaryServerAddress := net.JoinHostPort(hostName, primaryServerPort)
	requiringServerAddress := net.JoinHostPort(hostName, requiringServerPort)

	go primaryServer(network, primaryServerAddress)
	go requiringServer(network, requiringServerAddress)

	var wg sync.WaitGroup
	go client(&wg)

	defer wg.Wait()
}
