package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"strconv"
	"time"

	"github.com/MTthoas/GoApp29/functions"
)

const (
	startPort     = 1024
	endPort       = 8200
	timeout       = 2 * time.Second
	maxGoroutines = 5000
	username      = "zdeadzaz"
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

type SubmitChallenge struct {
    User    string `json:"User"`
    Secret  string `json:"Secret"`
    Content struct {
        Level     uint `json:"Level"`
        Challenge struct {
            Username string `json:"Username"`
            Secret   string `json:"Secret"`
            Points   uint   `json:"Points"`
        } `json:"Challenge"`
        Protocol  string `json:"Protocol"`
        SecretKey string `json:"SecretKey"`
    } `json:"Content"`
}


type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
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
		fmt.Printf("Pong found on port %d\n", pongPort)
		address := fmt.Sprintf("http://%s:%d", ip, pongPort)
		// Ping the service
		pingWG := &sync.WaitGroup{}
		// pingSemaphore := make(chan struct{}, maxGoroutines)
		// shouldCreateUser := pingMultipleTimes(address, 10000, pingWG, pingSemaphore)
		shouldCreateUser := true
		pingWG.Wait()

		if shouldCreateUser {
			fmt.Println("Hint received to create a new user. Signing up...")
			signup(address) // Je vais ajouter cette fonction pour la clarté

			// Check if the user was created
			responseCheck := PostandFetchContentWithBody(address+"/check", []byte(`{"user": "`+username+`"}`))
			fmt.Println("Check response:", string(responseCheck))
			
			response := PostandFetchContentWithBody(address+"/getUserSecret", []byte(`{"user": "`+username+`"}`))
			fmt.Println("Secret response:", string(response))
			
			h := sha256.New()
			h.Write([]byte(username))
			secret := fmt.Sprintf("%x", h.Sum(nil))

			println("Secret:", secret)

			getUserPoints := PostandFetchContentWithBody(address+"/getUserPoints", []byte(`{"user":"`+username+`", "secret":"`+secret+`"}`))
			fmt.Println("Points response:", string(getUserPoints))

						
						splits := strings.Split(strings.TrimSpace(string(getUserPoints)), "\n")
			if len(splits) < 2 {
				fmt.Println("Unexpected format for user points")
				return
			}

			cleanUserPoints := splits[1]
			userPointsUint, err := strconv.ParseUint(cleanUserPoints, 10, 32)
			if err != nil {
				fmt.Println("Error parsing cleaned user points to uint:", err)
				return
			}



			hints := make([]string, 0) // Tableau pour stocker les indices uniques

			for i := 0; i < 50; i++ {
				getHintResponse := PostandFetchContentWithBody(address+"/iNeedAHint", []byte(`{"user":"`+username+`", "secret":"`+secret+`"}`))
				hintString := string(getHintResponse)

				hintPrefix := "Here you go, your random hint:"
				startIndex := strings.Index(hintString, hintPrefix)
				if startIndex != -1 {
					hint := strings.TrimSpace(hintString[startIndex+len(hintPrefix):])
					if !contains(hints, hint) {
						hints = append(hints, hint)
					}
				}
			}
			fmt.Println("\n")

			fmt.Println("\nUnique Hints:")
				for _, hint := range hints {
					fmt.Println(hint)
				}

			fmt.Println("\n")

			getChallenge := PostandFetchContentWithBody(address+"/enterChallenge", []byte(`{"user":"`+username+`", "secret":"`+secret+`"}`))

			fmt.Println("Challenge response:", string(getChallenge))


			getUserLevel := PostandFetchContentWithBody(address+"/getUserLevel", []byte(`{"user":"`+username+`", "secret":"`+secret+`"}`))
			fmt.Println("Level response:", string(getUserLevel))


						cleanUserLevel := strings.TrimSpace(strings.Replace(string(getUserLevel), "Level:", "", 1))
			userLevelUint, err := strconv.ParseUint(cleanUserLevel, 10, 32)
			if err != nil {
				fmt.Println("Error parsing cleaned user level to uint:", err)
				return
			}

			fmt.Println("Level response:", string(getUserLevel))

			challenge := SubmitChallenge{
				User:   username,
				Secret: secret,
				Content: struct {
					Level     uint `json:"Level"`
					Challenge struct {
						Username string `json:"Username"`
						Secret   string `json:"Secret"`
						Points   uint   `json:"Points"`
					} `json:"Challenge"`
					Protocol  string `json:"Protocol"`
					SecretKey string `json:"SecretKey"`
				}{
					Level: uint(userLevelUint+1),
					Challenge: struct {
						Username string `json:"Username"`
						Secret   string `json:"Secret"`
						Points   uint   `json:"Points"`
					}{
						Username: username,
						Secret:   secret,
						Points:   uint(userPointsUint),
					},
					Protocol:  "MD5",
					SecretKey: secret,
				},
			}

			challengeJson, err := json.Marshal(challenge)
			if err != nil {
				fmt.Println("Error marshalling SubmitChallenge:", err)
				return
			}
			
			fmt.Println("Challenge JSON:", string(challengeJson))
			
			submitResponse := PostandFetchContentWithBody(address+"/submitSolution", challengeJson)
			fmt.Println("Submit challenge response:", string(submitResponse))
			return
		}
		fmt.Println("Finished pinging.")

	} else {
		fmt.Println("Pong not found on scanned ports.")
	}

}

func calculateMD5(input string) string {
    hasher := md5.New()
    hasher.Write([]byte(input))
    hashInBytes := hasher.Sum(nil)
    return hex.EncodeToString(hashInBytes)
}


func readPasswordFile() (string, error) {
    content, err := ioutil.ReadFile("password.txt")
    if err != nil {
        return "", err
    }
    splitted := strings.Split(string(content), ":")
    if len(splitted) < 2 {
        return "", fmt.Errorf("invalid password format in password.txt")
    }
    password := strings.TrimSpace(splitted[1])
    return password, nil
}

func writePasswordFile(content string) error {
    return ioutil.WriteFile("password.txt", []byte(content), 0644)
}

func contains(arr []string, str string) bool {
    for _, a := range arr {
        if a == str {
            return true
        }
    }
    return false
}


func PostandFetchContentWithBody(address string, payload []byte)  []byte{
	resp, err := http.Post(address, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("Signup error:", err)
		return nil
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil
	}

	return responseBody
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

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	fmt.Printf("Signup response: %s\n", responseBody)
}

func pingMultipleTimes(address string, times int, wg *sync.WaitGroup, semaphore chan struct{}) bool {
	hintsChan := make(chan string, times)

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
