package oraft

type CommandApplier interface {
	ApplyCommand(op string, value []byte) interface{}
}
