package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

func Psave(funcName string, err error) {
	if err != nil {
		errorMessage := fmt.Sprintf("[%v] %v: %v\n", time.Now().Format("06/01/02 15:04:05"), funcName, err)

		fmt.Printf(errorMessage)

		f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println(err)
		}

		defer f.Close()

		if _, err := f.WriteString(errorMessage); err != nil {
			log.Println(err)
		}
	}
}
