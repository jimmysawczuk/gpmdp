# gpmdp

[![Go Report Card](https://goreportcard.com/badge/github.com/jimmysawczuk/gpmdp)](https://goreportcard.com/report/github.com/jimmysawczuk/gpmdp)

**gpmdp** is a command-line utility to access the API of [Google Play Music Desktop Player](https://github.com/MarshallOfSound/Google-Play-Music-Desktop-Player-UNOFFICIAL-). It's meant to be used in macros and AppleScripts, like this one I use for pausing my music when I lock my computer:

```applescript
tell application "Google Play Music Desktop Player"
	if it is running then
		do shell script "GPMDP_AUTH_KEY=my-auth-key gpmdp pause"
	end if
end tell
```

## Usage

Install gpmdp using Go get:

```bash
$ go get github.com/jimmysawczuk/gpmdp
```

Then start up Google Play Music Desktop Player, and run:

```
$ gpmdp auth
```

Enter the PIN as prompted, then copy the provided authentication key. It's used as an environment variable by gpmdp, so you can either store this in your `.bashrc`/`.zshrc` or prefix any commands you want to run with it, i.e. `GPMDP\_AUTH_KEY=my-auth-key gpmdp pause`.

Finally, you can run `gpmdp play`, `gpmdp pause`, `gpmdp toggleshuffle` or `gpmdp status`, all of which should be self explanatory. Running `gpmdp help` shows a usage screen.
