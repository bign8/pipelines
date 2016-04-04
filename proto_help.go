package pipelines

import "github.com/bign8/pipelines/utils"

// TODO: use statistical datastructure to guarantee GUID uniqueness

// New constructs new record based on a source record
func (r *Record) New(data string) *Record {
	return &Record{
		CorrelationID: r.CorrelationID,
		Guid:          utils.RandUint64(),
		Data:          data,
		Test:          r.Test,
	}
}

// AsTest constructs a copy of this record intended for test routing
func (r Record) AsTest() *Record {
	return &Record{
		CorrelationID: r.CorrelationID,
		Guid:          utils.RandUint64(),
		Data:          r.Data,
		Test:          true,
	}
}

// NewRecord constructs a completely new record
func NewRecord(data string) *Record {
	return &Record{
		CorrelationID: utils.RandUint64(),
		Guid:          utils.RandUint64(),
		Data:          data,
	}
}

// ServiceKey generates a worker address for the worker designed to execute this work
func (w Work) ServiceKey() string {
	return w.Service + "." + w.Key
}
