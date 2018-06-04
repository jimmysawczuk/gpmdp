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

## Installing a premade binary

Download the [relevant pre-compiled binary](https://github.com/jimmysawczuk/gpmdp/releases) for your system and copy it to a directory in your `$PATH`.

## Installing from source

You can install `gpmdp` using `go get` or by cloning it yourself. Via `go get`:
```bash
$ go get github.com/jimmysawczuk/gpmdp
```

Or, if you prefer to clone it yourself:
```bash
$ git clone https://github.com/jimmysawczuk/gpmdp.git $GOPATH/src/github.com/jimmysawczuk/gpmdp
$ cd $GOPATH/src/github.com/jimmysawczuk/gpmdp
$ go install
```

You may need to add this directory to your `$PATH`, or copy the binary into a standard location.

## Usage

Start up Google Play Music Desktop Player, and run:

```
$ gpmdp auth
```

Enter the PIN as prompted, then copy the provided authentication key. It's used as an environment variable by gpmdp, so you can either store this in your `.bashrc`/`.zshrc` or prefix any commands you want to run with it, i.e. `GPMDP_AUTH_KEY=my-auth-key gpmdp pause`.

Finally, you can run `gpmdp play`, `gpmdp pause`, `gpmdp toggleshuffle` or `gpmdp status`, all of which should be self explanatory. Running `gpmdp help` shows a usage screen.
