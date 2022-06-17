package logger

import (
	"fmt"
	"os"
	"time"
)

func Psave(funcName string, err error) {
	if err != nil {
		errorMessage := fmt.Sprintf("[%v] %v: %v\n", time.Now().Format("06/01/02 15:04:05"), funcName, err)

		fmt.Printf(errorMessage)

		os.WriteFile("../log.txt", []byte(errorMessage), 0)
	}
}
