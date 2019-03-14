package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"bytes"
)

// Server configuration strings
const hostName,
	primaryToClientServerPort,
	primaryToRequiringServerPort,
	requiringServerPort,
	network string = "localhost",
	"8000",
	"8002",
	"8001",
	"udp"

const udpAddrSolveErr,
	udpListenErr,
	acceptUDPErr,
	connReadErr,
	unmarshalErr,
	marshalErr,
	primaryDialErr,
	resolveUDPErr,
	connWriteErr,
	clientDialErr string = "solving the UDPAddr",
	"trying to listen to UDP packets",
	"trying to accept a UDP connection",
	"trying to read string from connection",
	"trying to unmarshal JSON",
	"try to marshal struct",
	"trying to dial the Primary Server",
	"trying to resolve UDP Address",
	"trying to write through connection",
	"trying to dial from the client"

const cldntDial,
	cldntReq,
	cldntRead,
	cldntUnmarshal,
	cldntWrite string = "Internal Error: Could not dial Primary Server",
	"Internal Error: Could not generate proper request to Primary Server (Marshal Error)",
	"Internal Error: Could not read response from Primary Server",
	"Internal Error: Could not parse response from Primary Server (Unmarshal Error)",
	"Internal Error: Could not write request to Primary Server"

type serviceRequest struct {
	ServiceName string `json:"sn"`
	Values      string `json:"vs"`
	EndConn     bool   `json:"ENDCONN"`
}

type serviceResponse struct {
	Values  string `json:"vs"`
	EndConn bool   `json:"ENDCONN"`
}

// Gera uma função para imprimir erros genericamente
func newGenericErrMsgr(serverType, network, address string) func(string, error) string {
	return func(errInstance string, err error) string {
		return fmt.Sprintf("There was an error in %s for the %s with network %s and address %s\nError was: %s", errInstance, serverType, network, address, err.Error())
	}
}

// Nesta aplicação, as funções primaryServer, requiringServer e client
// serão acionadas como goroutines independentes

// Servidor primário, oferece serviços ao cliente e ao servidor requerente
func primaryServer(network, addressRequiring, addressClient string) {
	serverType := "Primary Server"
	genericErrMsg := newGenericErrMsgr(serverType, network, addressRequiring + " " + addressClient)

	// Os serviços oferecidos são funções guardadas em um map
	services := map[string]func(string) string{
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
		"unrot13": func(toDecipher string) string {
			bytesToDecipher := []byte(toDecipher)

			for i, v := range bytesToDecipher {
				if v >= 65 && v <= 90 {
					bytesToDecipher[i] = (((v - 65 + 26) - 13) % 26) + 65
				} else if v >= 97 && v <= 122 {
					bytesToDecipher[i] = (((v - 97 + 26) - 13) % 26) + 97
				}
			}

			return string(bytesToDecipher[:])
		},
	}

	udpAddr, err := net.ResolveUDPAddr(network, addressRequiring)

	if err != nil {
		fmt.Println(genericErrMsg(udpAddrSolveErr, err))
		return
	}

	conn, err := net.ListenUDP(network, udpAddr)

	if err != nil {
		fmt.Println(genericErrMsg(udpListenErr, err))
		return
	}

	/*
	_, writeBackAddr, err := conn.ReadFromUDP([]byte(""))

	if err != nil {
		fmt.Println(genericErrMsg(resolveUDPErr, err))
		return
	}
	*/

	udpAddr2, err := net.ResolveUDPAddr(network, addressClient)

	if err != nil {
		fmt.Println(genericErrMsg(udpAddrSolveErr, err))
		return
	}

	conn2, err := net.ListenUDP(network, udpAddr2)

	if err != nil {
		fmt.Println(genericErrMsg(udpListenErr, err))
		return
	}

	/*
	_, writeBackAddr2, err := conn2.ReadFromUDP([]byte(""))

	if err != nil {
		fmt.Println(genericErrMsg(resolveUDPErr, err))
		return
	}
	*/

	for {
		message := make([]byte, 1024)
		_, writeBackAddr, err := conn.ReadFromUDP(message)

		message = bytes.Trim(message, "\x00")

		if err != nil {
			fmt.Println(genericErrMsg(resolveUDPErr, err))
			continue
		}

		var req serviceRequest
		err = json.Unmarshal(message, &req)

		if err != nil {
			fmt.Println(genericErrMsg(unmarshalErr, err))
			continue
		}

		response := serviceResponse{services[req.ServiceName](req.Values), true}

		res, err := json.Marshal(response)

		if err != nil {
			fmt.Println(genericErrMsg(marshalErr, err))
		}

		res = append(res, '\n')
		_, err = conn.WriteTo(res, writeBackAddr)

		if err != nil {
			// fmt.Println(writeBackAddr, writeBackAddr2)
			fmt.Println(genericErrMsg(connWriteErr, err))
			continue
		}

		// Alternando conexões
		connTemp := conn
		conn = conn2
		conn2 = connTemp

		/*
		writeBackAddrTemp := writeBackAddr
		writeBackAddr = writeBackAddr2
		writeBackAddr2 = writeBackAddrTemp
		*/
	}
}

// Servidor requerente, serve ao cliente mas requer serviços do primário
func requiringServer(network, address string) {
	serverType := "Requiring Server"
	genericErrMsg := newGenericErrMsgr(serverType, network, address)

	storedString := "DEFAULT"

	services := map[string]func(string) string{
		"store": func(toStore string) string {
			raddr, err := net.ResolveUDPAddr(network, net.JoinHostPort(hostName, primaryToRequiringServerPort))

			if err != nil {
				fmt.Println(genericErrMsg(resolveUDPErr, err))
			}

			conn, err := net.DialUDP(network, nil, raddr)

			if err != nil {
				fmt.Println(genericErrMsg(primaryDialErr, err))
				return cldntDial
			}

			request := serviceRequest{"rot13", toStore, false}

			req, err := json.Marshal(request)

			if err != nil {
				fmt.Println(genericErrMsg(marshalErr, err))
				return cldntReq
			}

			req = append(req, '\n')
			_, err = conn.Write(req)

			if err != nil {
				fmt.Println(genericErrMsg(connWriteErr, err))
				return cldntWrite
			}

			response, err := bufio.NewReader(conn).ReadString('\n')

			if err != nil {
				fmt.Println(genericErrMsg(connReadErr, err))
				return cldntRead
			}

			var res serviceResponse
			err = json.Unmarshal([]byte(response), &res)

			if err != nil {
				fmt.Println(genericErrMsg(unmarshalErr, err))
				return cldntUnmarshal
			}

			storedString = res.Values

			return res.Values
		},
	}

	laddr, err := net.ResolveUDPAddr(network, address)

	if err != nil {
		fmt.Println(genericErrMsg(resolveUDPErr, err))
	}

	conn, err := net.ListenUDP(network, laddr)

	if err != nil {
		fmt.Println(genericErrMsg(udpListenErr, err))
	}

	for {
		message := make([]byte, 1024)
		_, writeBackAddr, err := conn.ReadFromUDP(message)

		message = bytes.Trim(message, "\x00")

		if err != nil {
			fmt.Println(genericErrMsg(resolveUDPErr, err))
			continue
		}

		var req serviceRequest
		err = json.Unmarshal(message, &req)

		if err != nil {
			fmt.Println(genericErrMsg(unmarshalErr, err))
			continue
		}

		response := serviceResponse{services[req.ServiceName](req.Values), true}

		res, err := json.Marshal(response)

		if err != nil {
			fmt.Println(genericErrMsg(marshalErr, err))
			continue
		}

		res = append(res, '\n')
		_, err = conn.WriteTo([]byte(res), writeBackAddr)

		if err != nil {
			fmt.Println(genericErrMsg(connWriteErr, err))
			continue
		}
	}
}

// Cliente, faz requisições
func client(wg *sync.WaitGroup) {
	defer wg.Done()

	genericErrMsg := newGenericErrMsgr("the Client", "tcp", "localhost")

	testString := "This is a test string."

	raddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(hostName, requiringServerPort))

	if err != nil {
		fmt.Println(genericErrMsg(resolveUDPErr, err))
		return
	}

	conn, err := net.DialUDP("udp", nil, raddr)

	if err != nil {
		fmt.Println(genericErrMsg(clientDialErr, err))
		return
	}

	request := serviceRequest{"store", testString, true}
	req, err := json.Marshal(request)

	if err != nil {
		fmt.Println(genericErrMsg(marshalErr, err))
		return
	}

	req = append(req, '\n')
	_, err = conn.Write(req)

	if err != nil {
		fmt.Println(genericErrMsg(connWriteErr, err))
		return
	}

	response, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil {
		fmt.Println(genericErrMsg(connReadErr, err))
		return
	}

	var res serviceResponse
	err = json.Unmarshal([]byte(response), &res)

	if err != nil {
		fmt.Println(genericErrMsg(unmarshalErr, err))
		return
	}

	cipheredString := res.Values

	// TODO Request deciphering from Primary Server
	fmt.Println(cipheredString)

	request = serviceRequest{"unrot13", cipheredString, true}
	req, err = json.Marshal(request)

	if err != nil {
		fmt.Println(genericErrMsg(marshalErr, err))
		return
	}

	raddr, err = net.ResolveUDPAddr("udp", net.JoinHostPort(hostName, primaryToClientServerPort))

	if err != nil {
		fmt.Println(genericErrMsg(resolveUDPErr, err))
		return
	}

	conn, err = net.DialUDP("udp", nil, raddr)

	if err != nil {
		fmt.Println(genericErrMsg(clientDialErr, err))
		return
	}

	req = append(req, '\n')
	_, err = conn.Write(req)

	if err != nil {
		fmt.Println(genericErrMsg(connWriteErr, err))
		return
	}

	response, err = bufio.NewReader(conn).ReadString('\n')

	if err != nil {
		fmt.Println(genericErrMsg(connReadErr, err))
		return
	}

	// res já foi declarada
	err = json.Unmarshal([]byte(response), &res)

	if err != nil {
		fmt.Println(genericErrMsg(unmarshalErr, err))
		return
	}

	decipheredString := res.Values

	fmt.Println(decipheredString)
}

func main() {
	primaryToRequiringServerAddress := net.JoinHostPort(hostName, primaryToRequiringServerPort)
	primaryToClientServerAddress := net.JoinHostPort(hostName, primaryToClientServerPort)
	requiringServerAddress := net.JoinHostPort(hostName, requiringServerPort)

	go primaryServer(network, primaryToRequiringServerAddress, primaryToClientServerAddress)
	go requiringServer(network, requiringServerAddress)

	var wg sync.WaitGroup
	wg.Add(1)
	go client(&wg)

	defer wg.Wait()
}
