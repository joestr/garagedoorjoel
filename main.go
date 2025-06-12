package main

import (
"bufio"
"fmt"
"log"
"strings"
"time"
"os"
"os/signal"

"go.bug.st/serial"
"periph.io/x/conn/v3/gpio"
"periph.io/x/host/v3"
"periph.io/x/host/v3/rpi"
)

const (
	serialPortName = "/dev/serial0"
	relayActiveMillis = 750
)

func main() {
	authorizedCallers := [3]string{"43000000"}

// Initialize periph.io
if _, err := host.Init(); err != nil {
log.Fatalf("Failed to init periph: %v", err)
}
relay := rpi.P1_16 // GPIO23
if err := relay.Out(gpio.Low); err != nil {
log.Fatalf("Failed to set relay low: %v", err)
}

c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt)
go func(){
    <-c
    relay.Out(gpio.Low)
    os.Exit(1)
}()

// Open serial port
mode := &serial.Mode{BaudRate: 115200}
port, err := serial.Open(serialPortName, mode)
if err != nil {
	log.Fatalf("Error opening serial port: %v", err)
}
defer port.Close()
fmt.Fprintln(port, "AT+CLIP=1\r") // Enable caller ID

reader := bufio.NewReader(port)
fmt.Println("Listening for calls...")

for {
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Error reading from serial:", err)
		continue
	}
	line = strings.TrimSpace(line)
	fmt.Println(">", line)

	if strings.HasPrefix(line, "+CLIP:") {

		for i := 0; i < len(authorizedCallers); i++ {
			authorizedCaller := authorizedCallers[i]
			if (strings.Contains(line, authorizedCaller)) {
				fmt.Println("Authorized caller detected. Opening garage.")
				triggerRelay(relay)
			}
		}
		fmt.Println("Hang up")
		fmt.Fprintln(port, "ATH\r") // Hang up
	}
}

}

func triggerRelay(relay gpio.PinOut) {
relay.Out(gpio.High)
time.Sleep(relayActiveMillis * time.Millisecond)
relay.Out(gpio.Low)
}

