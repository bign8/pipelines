#! /bin/bash

read -r -d '' DOCKERFILE <<- EOM
	FROM scratch
	ADD main /
	CMD ["/main"]
EOM

# echo "$DOCKERFILE" > Dockerfile

# find go source directories (TODO: assert their package is main)
find */* -name "*.go" -print0 | xargs -0 -n1 dirname | sort -u | while read line; do
  echo "Processing Directory '$line'"
  pushd $line > /dev/null

  # Generate go source
  echo -en "\tGolang "
  CGO_ENABLED=0 GOOS=linux go build -v -installsuffix cgo -o main -v . 2>&1 | while read; do echo -n .; done
  echo " Done"

  # Generating docker container
  echo -en "\tDocker "
  echo "$DOCKERFILE" > Dockerfile
  docker build -t pipeline-web-"$line":latest . | while read; do echo -n .; done
  echo " Done"

  # Cleaning up assets
  echo -en "\tFinish "
  rm -v main Dockerfile | while read; do echo -n .; done
  echo " Done"

  popd > /dev/null
done
