package repository

type Status int

const (
	StatusUninitialized Status = iota
	StatusReady
	StatusKeyRequired
)
