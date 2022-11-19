package main

import (
	"net/url"
	"time"
)

type VerificationItem struct {
	Domain          url.URL `json:"domain"`
	VerificationKey string  `json:"verification_key"`
	Verified        bool
	WarningStamp    time.Time
	ExpireStamp     time.Time
}
