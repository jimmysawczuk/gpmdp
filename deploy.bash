#!/bin/bash

APPNAME="gpmdp"
VERSION="0.1.0"

# Setup
mkdir -p deploy
echo "#!/bin/bash" > go-env.sh
go env >> go-env.sh;
source go-env.sh;

echo "Host OS:" $GOHOSTOS;
echo "Host Architecture:" $GOHOSTARCH;
echo ""

# Build
for GOOS in windows darwin linux; do
	for GOARCH in amd64 386; do
		exe=""
		if [[ $GOOS == "windows" ]]; then
			exe=".exe"
		fi

		echo "$GOOS/$GOARCH" && echo "----------------------------";
		GOOS=$GOOS GOARCH=$GOARCH go build -v -o $APPNAME github.com/jimmysawczuk/gpmdp

		# if this is the host OS/arch, the exe is put in the root of bin rather than a subdirectory
		if [[ $GOOS == $GOHOSTOS ]] && [[ $GOARCH == $GOHOSTARCH ]]; then
			mv $APPNAME deploy/$APPNAME-v$VERSION-${GOOS}-${GOARCH}$exe
		else
			mv $APPNAME deploy/$APPNAME-v$VERSION-${GOOS}-${GOARCH}$exe
		fi
		echo "";
	done
done

# Cleanup
rm go-env.sh