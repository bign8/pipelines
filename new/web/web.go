// Package web contains the configurations for the pipelines web example
package web

//go:generate gen.sh

import pipelines "github.com/bign8/pipelines/new"

// Streams available within the web crawler application
const (
	StreamINDEX pipelines.Stream = "index"
	StreamSTORE pipelines.Stream = "store"
	StreamCRAWL pipelines.Stream = "crawl"
)

// Types available within the web crawler application
const (
	TypeADDR pipelines.Type = "addr" // string URL
	TypeDUMP pipelines.Type = "dump" // byte dump of request
)

// type serializer struct{}
//
// func (*serializer) Serialize(obj interface{}) ([]byte, error) {
// 	return nil, nil
// }
//
// func (*serializer) Deserialize(bits []byte) (interface{}, error) {
// 	return nil, nil
// }
//
// func init() {
// 	pipelines.RegisterSerializer(&serializer{})
// }
