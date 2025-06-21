package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"strings"
	"time"

	"go.bug.st/serial"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/rpi"
)

const (
	serialPortName    = "/dev/serial0"
	relayActiveMillis = 750
)

type pinLevelMessage struct {
	State gpio.Level
	Reset gpio.Level
}

func setupGPIOInput(pinName string, levelChan chan pinLevelMessage) (gpio.PinIO, error) {
	log.Printf("Loading periph.io drivers")
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	// Find Pin by name
	p := gpioreg.ByName(pinName)

	// Configure Pin for input, configure pull as needed
	// Edge mode is currently not supported
	if err := p.In(gpio.PullNoChange, gpio.NoEdge); err != nil {
		return nil, err
	}

	// Setup Input signalling
	go func() {
		lastLevel := p.Read()
		// How often to poll levels, 100-150ms is fairly responsive unless
		// button presses are very fast.
		// Shortening the polling interval <100ms significantly increases
		// CPU load.
		for range time.Tick(100 * time.Millisecond) {
			currentLevel := p.Read()
			//log.Printf("level: %v", currentLevel)

			if currentLevel != lastLevel {
				levelChan <- pinLevelMessage{State: currentLevel, Reset: !currentLevel}
				lastLevel = currentLevel
			}
		}
	}()
	return p, nil
}

func main() {
	authorizedCallers := [3]string{"4300000"}

	// Initialize periph.io
	if _, err := host.Init(); err != nil {
		log.Fatalf("Failed to init periph: %v", err)
	}

	relay := rpi.P1_16 // GPIO23
	if err := relay.Out(gpio.Low); err != nil {
		log.Fatalf("Failed to set relay low: %v", err)
	}

	// Channel for communicating Pin levels
	levelChan := make(chan pinLevelMessage)

	p, err := setupGPIOInput("GPIO24", levelChan)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		relay.Out(gpio.Low)
		p.Out(gpio.Low)
		os.Exit(1)
	}()

	// Open serial port
	mode := &serial.Mode{BaudRate: 115200}
	port, err := serial.Open(serialPortName, mode)
	if err != nil {
		log.Fatalf("Error opening serial port: %v", err)
	}
	defer port.Close()

        reader := bufio.NewReader(port)

				log.Printf("Switch from data mode to command mode")
				fmt.Fprintln(port, "+++\r") /* lay ppp to background */
				line, err := reader.ReadString('\n')
				log.Println(line)

				log.Printf("Enabling caller ID")
				fmt.Fprintln(port, "AT+CLIP=1\r") // Enable caller ID
				line, err = reader.ReadString('\n')
				log.Println(line)
				
				log.Printf("Return back to data mode")
				fmt.Fprintln(port, "ATO\r") /* resume background ppp */    
				line, err = reader.ReadString('\n')
				log.Println(line)

	fmt.Println("Listening for calls...")

	for {
		select {
		case msg := <-levelChan:
			if msg.State {
			  log.Printf("Pin %s is high, nothing to do")
			} else if msg.Reset {
				log.Printf("Pin %s is low, someone is calling", p.Name())

				log.Printf("Switch from data mode to command mode")
				fmt.Fprintln(port, "+++\r") /* lay ppp to background */

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
							if strings.Contains(line, authorizedCaller) {
								fmt.Println("Authorized caller detected. Opening garage.")
								triggerRelay(relay)
							}
						}

						fmt.Println("Hang phone call up")
						fmt.Fprintln(port, "ATH\r") // Hang up
						

				                log.Printf("Return back to data mode")
				                fmt.Fprintln(port, "ATO\r") /* resume background ppp */

						break
					}

					
				}
			}
		default:
			// Any other ongoing tasks
		}
	}

}

func triggerRelay(relay gpio.PinOut) {
	relay.Out(gpio.High)
	time.Sleep(relayActiveMillis * time.Millisecond)
	relay.Out(gpio.Low)
}
