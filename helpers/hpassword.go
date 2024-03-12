package helpers

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"math/rand"
	"strconv"
	s "strings"
	"time"

	"github.com/nikitamirzani323/wigo_engine_invoice/configs"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
	}
	return string(hash)
}

func CheckPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return false, err
	}

	return true, nil
}
func HashPasswordMD5(password string) string {
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}
func Encryption(datatext string) (string, int) {
	min := 0
	max := 20 //149
	rand.Seed(time.Now().UnixNano())
	// keymap := rand.Intn(max-min) + min
	keymap := rand.Intn(max-min) + min
	var key string = configs.Keymap[keymap]
	var source string = configs.Sourcechar
	result := ""
	for i := 0; i < len(datatext); i++ {
		temp_indexsource := s.Index(source, string(datatext[i]))
		temp_indexkey := s.Index(key, string(key[temp_indexsource]))
		result += string(key[temp_indexkey])
	}
	return result, keymap
}
func Decryption(dataencrypt string) string {
	temp := s.Split(dataencrypt, "|")
	keymap, _ := strconv.Atoi(temp[1])
	var key string = configs.Keymap[keymap]
	var source string = configs.Sourcechar
	result := ""
	for i := 0; i < len(temp[0]); i++ {
		temp_indexkey := s.Index(key, string(dataencrypt[i]))
		temp_indexsource := s.Index(source, string(source[temp_indexkey]))
		result += string(source[temp_indexsource])
	}
	return result
}

func Parsing_Decry(data, pemisah string) (string, string) {
	temp_client := s.Split(data, pemisah)
	client_username := temp_client[0]
	client_rule := temp_client[1]
	return client_username, client_rule
}
