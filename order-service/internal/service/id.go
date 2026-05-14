package service

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s%x", strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000"), ".", ""), b)
}
