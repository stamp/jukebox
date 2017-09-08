package arduino

import (
	"fmt"
	"net"
	"sync"
	"time"

	"../player"

	"github.com/Sirupsen/logrus"
	cobs "github.com/dgryski/go-cobs"

	serial "go.bug.st/serial.v1"
	"go.bug.st/serial.v1/enumerator"
)

type Arduino struct {
	mode *serial.Mode

	Player *player.Player `inject:""`

	port serial.Port

	playing         int
	queue           []int
	toggle          bool
	toggleTimestamp time.Time

	r []byte
	g []byte
	b []byte

	sync.RWMutex
}

func (a *Arduino) Start() error {
	a.mode = &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	a.r = make([]byte, 0)
	a.g = make([]byte, 0)
	a.b = make([]byte, 0)

	go func() {
		for {
			err := a.Connect()
			logrus.Warn(err)
			<-time.After(time.Second)
		}
	}()

	go func() {
		for {
			err := a.equalizerSocket()
			logrus.Warn(err)
			<-time.After(time.Second)
		}
	}()

	a.Player.OnPlaylistChange(func(playing int, queue []int) {
		a.Lock()
		a.playing = playing
		a.queue = queue
		a.Unlock()

	})

	return nil
}

func (a *Arduino) Connect() error {
	err, portName := findPort()
	if err != nil {
		return err
	}

	a.Lock()
	port, err := serial.Open(portName, a.mode)
	a.Unlock()
	if err != nil {
		return err
	}
	a.port = port

	logrus.Infof("Connected to arduino tru %s", portName)

	firstMessage := make(chan struct{})
	// Writer
	go func() {
		<-firstMessage
		for {
			<-time.After(time.Second / 30)
			err := a.WriteLights()
			if err != nil {
				logrus.Error(err)
				return
			}
		}
	}()

	// Reader
	readbuff := make([]byte, 100)
	buff := make([]byte, 0)

	decoder := cobs.New()

	for {
		// Reads up to 100 bytes
		n, err := port.Read(readbuff)
		if firstMessage != nil {
			close(firstMessage)
			firstMessage = nil
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("EOF")
		}

		buff = append(buff, readbuff[:n]...)

		lastFound := 0
		start := 0
		for k, v := range buff {
			if v == 0 {
				lastFound = k + 1
				packet := buff[start:k]
				start = k

				data, err := decoder.Decode(packet)

				if len(data) == 1 {
					go a.Player.Queue(int(data[0]) - 1)
				} else {
					logrus.Warnf("Unkown data: %x|%x => %s | %x", packet, data, err, buff)
				}
			}
		}

		buff = buff[lastFound:]

		//if n == 4 && buff[0] == 0x99 {
		//logrus.Infof("Button %d", buff[1])
		//go a.Player.Queue(int(buff[1]) - 1)
		//} else {
		//logrus.Warnf("Unkown data: %x", string(buff[:n]))
		//}
	}

	return nil
}

func (a *Arduino) equalizerSocket() error {
	c, err := net.Dial("unix", "/tmp/led-strip.sock")
	if err != nil {
		return err
	}

	buf := make([]byte, 4)
	pointer := byte(0)
	for {
		_, err := c.Read(buf[:])
		if err != nil {
			return err
		}

		if len(buf) == 4 {
			if pointer > buf[0] {
				pointer = 0
			} else {
				pointer = buf[0]
				for int(pointer) >= len(a.r) {
					a.r = append(a.r, 0)
					a.g = append(a.g, 0)
					a.b = append(a.b, 0)
				}
				a.r[pointer] = buf[1]
				a.g[pointer] = buf[2]
				a.b[pointer] = buf[3]
			}
		}
	}
}

func setBit(n byte, pos uint, condition bool) byte {
	if condition {
		n |= (1 << pos)
	}
	return n
}

func (a *Arduino) WriteLights() error {
	lights := make([]bool, 40)

	a.RLock()
	p := a.playing
	q := a.queue
	t := a.toggle
	if a.toggleTimestamp.Before(time.Now()) {
		a.toggle = !a.toggle
		a.toggleTimestamp = time.Now().Add(time.Second / 4)
	}
	a.RUnlock()

	if p > 0 { // Current playing, flashing
		lights[p-1] = t
	}

	for _, v := range q {
		if v < 40 {
			lights[v] = true
		}
	}

	toSend := []byte{}

	for i := 0; i <= 4; i++ {
		var c byte

		c = setBit(c, 0, lights[0+i])
		c = setBit(c, 1, lights[5+i])
		c = setBit(c, 2, lights[10+i])
		c = setBit(c, 3, lights[15+i])
		c = setBit(c, 4, lights[20+i])
		c = setBit(c, 5, lights[25+i])
		c = setBit(c, 6, lights[30+i])
		c = setBit(c, 7, lights[35+i])

		toSend = append(toSend, c)
	}

	for k, _ := range a.r {
		if k == 0 {
			continue
		}

		if k <= 156/2 {
			toSend = append(toSend, a.r[k])
			toSend = append(toSend, a.g[k])
			toSend = append(toSend, a.b[k])
		}
	}

	fmt.Printf("%d\r", len(toSend))
	encoder := cobs.New()
	toSend = encoder.Encode(toSend)

	toSend = append(toSend, 0) // Add packet end

	_, err := a.port.Write(toSend)
	return err
	return nil
}

func findPort() (error, string) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return err, ""
	}

	if len(ports) == 0 {
		return fmt.Errorf("No serial ports found!"), ""
	} else {
		for _, port := range ports {
			if port.IsUSB && port.VID == "1a86" && port.PID == "7523" {
				return nil, port.Name
			}
		}
	}

	return fmt.Errorf("Did not match VID:PID"), ""
}
