# Pipelines
Cloud based DAG infrastructure with inter-node release.

[![License MIT](https://img.shields.io/npm/l/express.svg)](http://opensource.org/licenses/MIT)
[![ReportCard](http://goreportcard.com/badge/bign8/pipelines)](http://goreportcard.com/report/bign8/pipelines)
[![Build Status](https://travis-ci.org/bign8/pipelines.svg?branch=master)](https://travis-ci.org/bign8/pipelines)
[![GoDoc](http://godoc.org/github.com/bign8/pipelines?status.png)](http://godoc.org/github.com/bign8/pipelines)
[![GitHub release](http://img.shields.io/github/release/bign8/pipelines.svg)](https://github.com/bign8/pipelines/releases)


## POC Instructions

```sh
$ gnatsd -m 8222
$ nats-top
$ export PATH=$PATH:$GOPATH/bin
$ go install ./...
$ pipeline server
$ pipeline agent  # As many agents as you would like
$ pipeline load sample/web
$ pipeline send crawl_request https://en.wikipedia.org/wiki/Main_page
```
