package service

import (
	"crypto/rand"
	"encoding/hex"

)

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}


