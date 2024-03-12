package helpers

import (
	"crypto/rand"
	"io"
)

func GetEndRangeDate(month string) string {
	end := ""
	switch month {
	case "JAN":
		end = "31"
	case "FEB":
		end = "28"
	case "MAR":
		end = "31"
	case "APR":
		end = "30"
	case "MAY":
		end = "31"
	case "JUN":
		end = "30"
	case "JUL":
		end = "31"
	case "AUG":
		end = "31"
	case "SEP":
		end = "30"
	case "OCT":
		end = "31"
	case "NOV":
		end = "30"
	case "DEC":
		end = "31"
	}
	return end
}

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func GenerateNumber(max int) string {
	b := make([]byte, max)
	n, err := io.ReadAtLeast(rand.Reader, b, max)
	if n != max {
		panic(err)
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b)
}
