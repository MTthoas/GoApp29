package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/MTthoas/GoApp29/functions"
)

const (
	startPort     = 1024
	endPort       = 65535
	timeout       = 2 * time.Second
	maxGoroutines = 5000
	username      = "matthiasV3"
)

type SignupRequest struct {
	User string `json:"user"`
}

type SecretResponse struct {
	Secret string `json:"secret"`
}

type LevelRequest struct {
	User   string `json:"user"`
	Secret string `json:"secret"`
}

func main() {
	ip := "10.49.122.144"
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxGoroutines)
	pongChan := make(chan int)

	// Scan all ports to find the "Pong" service
	go func() {
		for port := startPort; port <= endPort; port++ {
			wg.Add(1)
			go functions.ScanPort(ip, port, &wg, semaphore, pongChan)
		}
		wg.Wait()
		close(pongChan)
	}()

	pongPort, ok := <-pongChan
	if ok {
		fmt.Printf("Pong found on port %d.\n", pongPort)
		address := fmt.Sprintf("http://%s:%d", ip, pongPort)
		// Ping the service
		pingWG := &sync.WaitGroup{}
		pingSemaphore := make(chan struct{}, maxGoroutines)
		shouldCreateUser := pingMultipleTimes(address, 10000, pingWG, pingSemaphore)
		pingWG.Wait()

		if shouldCreateUser {
			fmt.Println("Hint received to create a new user. Signing up...")
			signup(address) // Je vais ajouter cette fonction pour la clarté
			return
		}
		fmt.Println("Finished pinging.")

	} else {
		fmt.Println("Pong not found on scanned ports.")
	}
}

func signup(address string) {
	signupPayload := SignupRequest{User: username}
	payloadBytes, _ := json.Marshal(signupPayload)
	resp, err := http.Post(fmt.Sprintf("%s/signup", address), "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Signup error:", err)
		return
	}
	defer resp.Body.Close()

	// Lire le body de la réponse
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Afficher le contenu du body
	fmt.Printf("Signup response: %s\n", responseBody)

	// Vous pouvez ajouter des logiques supplémentaires pour traiter la réponse si nécessaire
	// Par exemple, déterminer si l'inscription a réussi ou échoué en fonction de la réponse
}

func pingMultipleTimes(address string, times int, wg *sync.WaitGroup, semaphore chan struct{}) bool {
	hintsChan := make(chan string, times) // Canal pour capturer tous les indices

	for i := 0; i < times; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			resp, err := http.Get(fmt.Sprintf("%s/ping", address))
			if err != nil {
				fmt.Printf("Ping error: %s\n", err.Error())
				return
			}
			defer resp.Body.Close()

			// Lire et traiter l'indice
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			hintsChan <- string(bodyBytes)
		}()
	}

	// Attente de la fin de tous les goroutines
	wg.Wait()
	close(hintsChan)

	// Parcourir les indices
	shouldCreateUser := false
	for hint := range hintsChan {
		// fmt.Println("Received hint:", hint)
		if hint == "pong... ping pong... Maybe you should create a new user..." {
			shouldCreateUser = true
			break
		}
	}
	return shouldCreateUser
}
