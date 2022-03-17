package service

import "time"

// Clock is system clock.
type Clock interface {

	// Now returns current time.
	Now() time.Time
}

// ClockFunc is functional interface of Clock.
type ClockFunc func() time.Time

func (f ClockFunc) Now() time.Time {
	return f()
}

// IDGenerator generates unique IDs.
type IDGenerator interface {

	// GenerateID return unique ID.
	GenerateID() string
}

// IDGeneratorFunc is functional interface of IDGenerator.
type IDGeneratorFunc func() string

func (f IDGeneratorFunc) GenerateID() string {
	return f()
}
