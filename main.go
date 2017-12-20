package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type writePayload struct {
	Namespace string        `json:"namespace"`
	Method    string        `json:"method"`
	Arguments []interface{} `json:"arguments"`
	RequestID int           `json:"requestID"`
}

type readPayloadOrWriteReturn struct {
	Channel string      `json:"channel"`
	Payload interface{} `json:"payload"`

	Namespace string      `json:"namespace"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	RequestID int         `json:"requestID"`
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
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"time"`
	Track struct {
		Album  string `json:"album"`
		Artist string `json:"artist"`
		Title  string `json:"title"`
	} `json:"track"`
	Volume int64 `json:"volume"`
}

var ws *websocket.Conn
var playerState state

var stateReady map[string]bool
var readyCh chan bool
var stateChangeCh chan bool
var authCh chan interface{}
var inittedCh chan bool

func main() {
	authCh = make(chan interface{})
	stateChangeCh = make(chan bool)
	readyCh = make(chan bool)
	inittedCh = make(chan bool)
	stateReady = map[string]bool{}

	var err error
	ws, _, err = websocket.DefaultDialer.Dial("ws://localhost:5672", nil)
	if err != nil {
		log.Fatalf("couldn't connect to gpmdp: %s", err)
	}
	defer ws.Close()

	go listen()
	go waitForInit()
	<-readyCh

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

	case "status":
		err = status()

	case "toggleshuffle":
		err = toggleShuffle()

	case "togglerepeat":
		err = toggleRepeat()

	case "next":
		err = next()

	case "prev":
		err = prev()

	default:
		err = errors.New("invalid subcommand:" + os.Args[1])
	}

	if err != nil {
		fmt.Println(err)
		usage()
		os.Exit(1)
	}
}

func listen() error {
	for {
		var target readPayloadOrWriteReturn
		err := ws.ReadJSON(&target)
		if err != nil {
			log.Printf("listen err: %s", err)
			os.Exit(2)
			return err
		}

		switch target.Namespace {
		case "result":
			stateChangeCh <- true
			continue
		}

		switch target.Channel {
		case "API_VERSION":
			marshalData(&playerState.APIVersion, target.Payload)
			stateReady[target.Channel] = true
			readyCh <- true

		case "playState":
			marshalData(&playerState.PlayState, target.Payload)
			stateReady[target.Channel] = true

		case "volume":
			marshalData(&playerState.Volume, target.Payload)
			stateReady[target.Channel] = true

		case "shuffle":
			marshalData(&playerState.Shuffle, target.Payload)
			stateReady[target.Channel] = true

		case "repeat":
			marshalData(&playerState.Repeat, target.Payload)
			stateReady[target.Channel] = true

		case "track":
			marshalData(&playerState.Track, target.Payload)
			stateReady[target.Channel] = true

		case "rating":
			marshalData(&playerState.Rating, target.Payload)
			stateReady[target.Channel] = true

		case "time":
			marshalData(&playerState.Time, target.Payload)
			stateReady[target.Channel] = true

		case "library", "lyrics", "playlists", "queue", "search-results", "settings:theme", "settings:themeColor", "settings:themeType":
			continue

		case "connect":
			authCh <- target.Payload

		default:
			log.Println("unhandled channel:", target.Channel)
			continue
		}
	}
}

func waitForInit() {
	for len(stateReady) < 8 {
		time.Sleep(100 * time.Millisecond)
	}

	inittedCh <- true
	close(inittedCh)
}

func marshalData(dst interface{}, src interface{}) {
	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(src)
	json.NewDecoder(buf).Decode(dst)
}

func setupAuth() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "connect",
		Method:    "connect",
		Arguments: []interface{}{"Applescript Invoker"},
	})
	if err != nil {
		return errors.Wrap(err, "auth: connect")
	}

	authResponse := <-authCh

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
		RequestID: 2,
	})

	authResponse = <-authCh
	resp, ok := authResponse.(string)
	if !ok {
		return errors.Errorf("invalid response received: %v", authResponse)
	}

	if resp == "CODE_REQUIRED" {
		return errors.New("invalid PIN entered")
	}

	fmt.Println("GPMDP_AUTH_KEY=" + resp)
	os.Setenv("GPMDP_AUTH_KEY", resp)

	return nil
}

func authenticate() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "connect",
		Method:    "connect",
		Arguments: []interface{}{"Applescript Invoker", os.Getenv("GPMDP_AUTH_KEY")},
		RequestID: 1,
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

	<-stateChangeCh

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

	<-stateChangeCh

	return nil
}

func toggleRepeat() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "playback",
		Method:    "toggleRepeat",
		Arguments: []interface{}{},
		RequestID: 2,
	})
	if err != nil {
		return errors.Wrap(err, "toggleRepeat")
	}

	<-stateChangeCh

	return nil
}

func prev() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "playback",
		Method:    "rewind",
		Arguments: []interface{}{},
		RequestID: 2,
	})
	if err != nil {
		return errors.Wrap(err, "prev")
	}

	<-stateChangeCh

	return nil
}

func next() error {
	err := ws.WriteJSON(writePayload{
		Namespace: "playback",
		Method:    "forward",
		Arguments: []interface{}{},
		RequestID: 2,
	})
	if err != nil {
		return errors.Wrap(err, "next")
	}

	<-stateChangeCh

	return nil
}

func status() error {
	if res, ok := <-inittedCh; !res && ok {
		return errors.New("never initted")
	}

	if playerState.PlayState {
		fmt.Printf(`Currently playing:
	Track: %s
	Artist: %s
	Album: %s
	Time: %s / %s
`,
			playerState.Track.Title,
			playerState.Track.Artist,
			playerState.Track.Album,
			time.Duration(playerState.Time.Current*1e6),
			time.Duration(playerState.Time.Total*1e6),
		)
	} else {
		fmt.Println("Playback paused")
	}
	return nil
}

func usage() {
	fmt.Println("Usage: gpmdp <command>")
	fmt.Println("Available commands:")
	fmt.Println("  auth: authenticates app so it can control GPMDP")
	fmt.Println("  next: advance to the next song")
	fmt.Println("  pause: pauses playback")
	fmt.Println("  play: resumes playback")
	fmt.Println("  prev: return to the previous song")
	fmt.Println("  status: shows currently playing track")
	fmt.Println("  togglerepeat: toggles repeat mode")
	fmt.Println("  toggleshuffle: toggles shuffle mode")
	fmt.Println("")
}
