#! /bin/bash
# Thanks: https://blog.codeship.com/building-minimal-docker-containers-for-go-applications/

set -e # causes the whole script to fail if any commands fail
set -o pipefail # causes pipes to fail if the input command fails

cat > Dockerfile <<- EOM
	FROM scratch
	ADD ca-certificates.crt /etc/ssl/certs/
	ADD main /
	CMD ["/main"]
EOM

echo -n "Downloading CA Certs "
curl https://curl.haxx.se/ca/cacert.pem -# -o certz.crt 2>&1 | xxd | while read; do echo -n .; done
echo " Done"

# find go source directories (TODO: assert their package is main)
find */* -name "*.go" -print0 | xargs -0 -n1 dirname | sort -u | while read line; do
  echo "Processing Directory '$line'"
  pushd $line > /dev/null
  cp ../certz.crt ca-certificates.crt
  cp ../Dockerfile Dockerfile

  # Generate go source
  echo -en "\tGolang "
  CGO_ENABLED=0 GOOS=linux go build -v -installsuffix cgo -o main -v . 2>&1 | while read; do echo -n .; done
  echo " Done"

  # Generating docker container
  echo -en "\tDocker "
  docker build -t pipeline-web-"$line":latest . | while read; do echo -n .; done
  echo " Done"

  # Cleaning up assets
  echo -en "\tFinish "
  rm -v main Dockerfile ca-certificates.crt | while read; do echo -n .; done
  echo " Done"

  popd > /dev/null
done

# Cleanup files
rm certz.crt Dockerfile
echo "Complete"
