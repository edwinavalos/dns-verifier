package utils

import (
	"math/rand"
	"sync"
)

func SyncMap2Map(syncMap *sync.Map) map[string]interface{} {
	regMap := make(map[string]interface{})
	if syncMap != nil {
		syncMap.Range(func(k interface{}, v interface{}) bool {
			regMap[k.(string)] = v
			return true
		})
	}
	return regMap
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
