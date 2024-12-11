package entity

import (
	"github.com/go-playground/validator/v10"
	"sync"
)

var (
	validate *validator.Validate
	initOnce sync.Once
)

func init() {
	initOnce.Do(func() {
		validate = validator.New()
	})
}
