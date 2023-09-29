package functions

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	startPort     = 1
	endPort       = 65535
	timeout       = 2 * time.Second
	maxGoroutines = 5000 // Nombre maximal de goroutines actives simultanément
)

func ScanPort(ip string, port int, wg *sync.WaitGroup, semaphore chan struct{}, pongChan chan int) {
	defer wg.Done()

	semaphore <- struct{}{}        // Acquérir un jeton
	defer func() { <-semaphore }() // Relâcher un jeton

	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return
	}
	defer conn.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s/ping", address))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if string(body) == "pong" {
		fmt.Printf("Port %d est ouvert et a renvoyé 'pong'\n", port)
		pongChan <- port // Send the port number to the pongChan
		return
	}
}