package tests

import (
	"database/sql/driver"
	"time"
)

type AnyInt struct{}
func (a AnyInt) Match(v driver.Value) bool {
	_, ok := v.(int64)
	return ok
}

type AnyString struct{}
func (a AnyString) Match(v driver.Value) bool {
	_, ok := v.(string)
	return ok
}

type AnyTime struct{}
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok	
}
