package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/praserx/ipconv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type CurrConfig struct {
	Port              string
	IsMaster          bool
	IsPreferredMaster bool
	IsPeerAvailable bool
	Peer              string
	Local             string
	Vlan              string
}

var cfg *CurrConfig

const interval = 5

//Why need this ?
const mutexLocked = 1

func MutexLocked(m *sync.Mutex) bool {
	state := reflect.ValueOf(m).Elem().FieldByName("state")
	return state.Int()&mutexLocked == mutexLocked
}

func (c *CurrConfig) Task(m *sync.Mutex) {
	if MutexLocked(m) {
		return
	}
	m.Lock()
	defer m.Unlock()
  c.PeerAvailable()
	log.Trace().Msg(strconv.Itoa(interval) + " sec task")
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 6 {
		fmt.Println(os.Args[0]+" Ethernet0 10.10.0.10 10.10.0.20 100 [--trace]")
		return
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg = &CurrConfig{}

	if len(os.Args) == 6 && os.Args[5] == "--trace" {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}

	if len(os.Args) >= 5 {
		cfg.Port = os.Args[1]
		cfg.Local = os.Args[2]
		cfg.Peer = os.Args[3]
		cfg.Vlan = os.Args[4]
	}

  err := runServer(cfg.Local)
  if err !=nil {
    fmt.Println(err.Error())
    return
  }

	cfg.ColdStart()

	var m sync.Mutex
	s := gocron.NewScheduler(time.UTC)
	s.Every(interval).Seconds().Do(cfg.Task, &m)
	s.StartBlocking()
}

func (c *CurrConfig) ColdStart() {
	lc := net.ParseIP(c.Local)
	lcInt, err := ipconv.IPv4ToInt(lc)
	if err != nil {
		return
	}
	peer := net.ParseIP(c.Peer)
	peerInt, err := ipconv.IPv4ToInt(peer)
	if err != nil {
		return
	}
	if lcInt > peerInt {
		log.Trace().Msg("Local preferred to be master")
		c.IsPreferredMaster = true
	} else {
		log.Trace().Msg("Local preferred to be slave")
		c.IsPreferredMaster = false
	}

	if c.IsPreferredMaster {
		c.IsMaster = true
    c.SetVlanAsMaster()
	} else {
		c.IsMaster = false
		c.RemoveVlanAsMaster()
  }

}

func (c *CurrConfig) PeerAvailable() bool {
	//err := checkHostAvail(c.Peer)
  err := rpcPing(c.Peer)
	if err != nil {
		return false
	}
	return true
}

func (c *CurrConfig) SetVlanAsMaster() {
	log.Trace().Msg("SetVlanAsMaster")
}
func (c *CurrConfig) RemoveVlanAsMaster() {
	log.Trace().Msg("RemoveVlanAsMaster")
}
