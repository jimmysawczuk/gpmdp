package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type writePayload struct {
	Namespace string        `json:"namespace"`
	Method    string        `json:"method"`
	Arguments []interface{} `json:"arguments"`
	RequestID int           `json:"requestID"`
}

type writeReturn struct {
	Namespace string      `json:"namespace"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	RequestID int         `json:"requestID"`
}

type readPayload struct {
	Channel string      `json:"channel"`
	Payload interface{} `json:"payload"`
}

type state struct {
	APIVersion string `json:"API_VERSION"`
	PlayState  bool   `json:"playState"`
	Rating     struct {
		Disliked bool `json:"disliked"`
		Liked    bool `json:"liked"`
	} `json:"rating"`
	Repeat  string `json:"repeat"`
	Shuffle string `json:"shuffle"`
	Time    struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"time"`
	Track struct {
		Album  string `json:"album"`
		Artist string `json:"artist"`
		Title  string `json:"title"`
	} `json:"track"`
}

var ws *websocket.Conn
var requestID = 1
var playerState state

func main() {
	var err error
	ws, _, err = websocket.DefaultDialer.Dial("ws://localhost:5672", nil)
	if err != nil {
		log.Fatalf("couldn't connect to gpmdp: %s", err)
	}
	defer ws.Close()

	readInitialState()

	if auth := os.Getenv("GPMDP_AUTH_KEY"); auth != "" && os.Args[1] != "auth" {
		authenticate()
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "auth":
		err = setupAuth()

	case "pause":
		err = pause()

	case "play":
		err = play()

	case "toggleshuffle":
		err = toggleShuffle()

	default:
		err = errors.New("invalid subcommand:" + os.Args[1])
	}

	if err != nil {
		fmt.Println(err)
		usage()
		os.Exit(1)
	}
}

func readInitialState() error {
	tempState := map[string]interface{}{}

	for {
		target := readPayload{}
		err := ws.ReadJSON(&target)
		if err != nil {
			break
		}

		switch target.Channel {
		case "library", "lyrics", "search-results", "settings:theme", "settings:themeColor", "settings:themeType":
			// do nothing for these
		default:
			tempState[target.Channel] = target.Payload
		}

		if target.Channel == "library" {
			break
		}
	}

	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(tempState)
	if err != nil {
		return errors.Wrap(err, "read initial state")
	}

	err = json.NewDecoder(buf).Decode(&playerState)
	if err != nil {
		return errors.Wrap(err, "read initial state")
	}

	return nil
}

func setupAuth() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "connect",
		Method:    "connect",
		Arguments: []interface{}{"Applescript Invoker"},
		RequestID: 1,
	})
	if err != nil {
		return errors.Wrap(err, "auth: connect")
	}

	authResponse := readPayload{}
	err = ws.ReadJSON(&authResponse)
	if err != nil {
		return errors.Wrap(err, "auth: handshake response")
	}

	pin := ""
	fmt.Printf("Enter a PIN: ")
	_, err = fmt.Scanln(&pin)
	if err != nil {
		return errors.Wrap(err, "auth: get pin")
	}

	err = ws.WriteJSON(writePayload{
		Namespace: "connect",
		Method:    "connect",
		Arguments: []interface{}{"Applescript Invoker", pin},
	})

	err = ws.ReadJSON(&authResponse)
	if err != nil {
		return errors.Wrap(err, "auth: read pin response")
	}

	payload := authResponse.Payload.(string)

	if payload == "CODE_REQUIRED" {
		return errors.Wrap(errors.New("bad pin"), "auth")
	}

	fmt.Println("GPMDP_AUTH_KEY=" + payload)
	os.Setenv("GPMDP_AUTH_KEY", payload)

	return nil
}

func authenticate() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "connect",
		Method:    "connect",
		Arguments: []interface{}{"Applescript Invoker", os.Getenv("GPMDP_AUTH_KEY")},
	})
	if err != nil {
		return errors.Wrap(err, "authenticate: connect")
	}

	return nil
}

func pause() error {
	if !playerState.PlayState {
		// nothing to pause; it's already paused or no music is playing
		return nil
	}

	err := togglePlayState()
	if err != nil {
		return errors.Wrap(err, "pause")
	}
	return nil
}

func play() error {
	if playerState.PlayState {
		// already playing
		return nil
	}

	err := togglePlayState()
	if err != nil {
		return errors.Wrap(err, "play")
	}
	return nil
}

func togglePlayState() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "playback",
		Method:    "playPause",
		Arguments: []interface{}{},
		RequestID: 2,
	})
	if err != nil {
		return errors.Wrap(err, "togglePlayState")
	}

	target := writeReturn{}
	err = ws.ReadJSON(&target)
	if err != nil {
		return errors.Wrap(err, "togglePlayState")
	}

	return nil
}

func toggleShuffle() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "playback",
		Method:    "toggleShuffle",
		Arguments: []interface{}{},
		RequestID: 2,
	})
	if err != nil {
		return errors.Wrap(err, "toggleShuffle")
	}

	target := writeReturn{}
	err = ws.ReadJSON(&target)
	if err != nil {
		return errors.Wrap(err, "toggleShuffle")
	}

	return nil
}

func usage() {
	fmt.Println("Usage: gpmdp <command>")
	fmt.Println("Available commands:")
	fmt.Println("  auth: authenticates app so it can control GPMDP")
	fmt.Println("  pause: pauses playback")
	fmt.Println("  play: resumes playback")
	fmt.Println("  toggleshuffle: toggles shuffle mode")
	fmt.Println("")
}
