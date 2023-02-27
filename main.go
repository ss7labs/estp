package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/keegancsmith/shell"
	"github.com/praserx/ipconv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CurrConfig struct {
	Port              string
	IsMaster          bool
	IsPreferredMaster bool
	IsPeerAvailable   bool
	IsIntAvailable    bool
	Peer              string
	Local             string
	Vlan              []string
}

var showCMD string
var sudoCMD string

var cfg *CurrConfig

const SHOW = "show"
const SUDO = "sudo"
const testSHOW = "./show"
const testSUDO = "./sudo"

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
	log.Trace().Msg(strconv.Itoa(interval) + " sec task")

	if !c.IsMaster {

		if c.PeerAvailable() {
			c.IsPeerAvailable = true
		} else {
			c.IsPeerAvailable = false
			log.Trace().Msg("Peer not available")
		}

	}
	if !c.IsMaster && !c.IsPeerAvailable {
		c.IsMaster = true
		log.Trace().Msg("Master in DOWN")
	}
	if c.IsMaster && !c.IsPreferredMaster && !c.IsPeerAvailable {
		if c.PeerAvailable() {
			c.IsPeerAvailable = true
			c.IsMaster = false
			log.Trace().Msg("Master Restored")
		}
	}
	if c.IsPreferredMaster {
		currState := c.IsIntAvail()
		if currState && c.IsIntAvailable {
			log.Trace().Msg("Master check interface state: All ok, don't touch vlan.")
			return
		}

		if !currState && !c.IsIntAvailable {
			log.Trace().Msg("Master check interface state: All is BAD, don't touch vlan.")
			return
		}

		if !currState && c.IsIntAvailable {
			log.Trace().Msg("Master check interface state: Interface DOWN, Change current state.")
			c.IsIntAvailable = false

			// Add Vlans to interface at slave via RPC
			log.Trace().Msg("Master check interface state: Interface DOWN, add vlans at slave.")
			err := rpcPing(c.Peer, "AddVlans")
			if err != nil {
				log.Trace().Msg("Master, failed to add vlans at slave.")
			}
			return
		}

		if currState && !c.IsIntAvailable {
			log.Trace().Msg("Master check interface state: Interface RESTORED, Change current state.")
			c.IsIntAvailable = true

			// Remove Vlans from interface at slave via RPC
			log.Trace().Msg("Master check interface state: Interface RESTORED, remove vlans from slave.")
			err := rpcPing(c.Peer, "RemoveVlans")
			if err != nil {
				log.Trace().Msg("Master, failed to remove vlans at slave.")
			}
			return
		}

	}

}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 6 {
		fmt.Println(os.Args[0] + " Ethernet0 10.10.0.10 10.10.0.20 100 [--trace]")
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
		cfg.Vlan = parseVlans(os.Args[4])
		fmt.Println(cfg.Vlan)
	}

	err := runServer(cfg)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

  showCMD = testSHOW
	sudoCMD = testSUDO

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
		c.RemoveVlansFromPort()
	}
	c.IsIntAvailable = c.IsIntAvail()
}

func (c *CurrConfig) IsIntAvail() bool {
	//out, err := shell.Commandf("echo %s  world   %s", "hello", "quote!me").Output()
	//show interfaces status Ethernet36
	cmd := showCMD + " interfaces status %s"

	out, err := shell.Commandf(cmd, c.Port).Output()
	if err != nil {
		log.Trace().Msg("show interface failed: " + err.Error())
		return false
	}

	portInfo := string(out)
	temp := strings.Split(portInfo, "\n")
	tmpFields := strings.Fields(temp[2])
	state := tmpFields[7]

	if state == "up" {
		return true
	}
	return false
}
func (c *CurrConfig) PeerAvailable() bool {
	//err := checkHostAvail(c.Peer)
	err := rpcPing(c.Peer, "")
	if err != nil {
		return false
	}
	return true
}

func (c *CurrConfig) SetVlanAsMaster() {
	log.Trace().Msg("SetVlanAsMaster")
	for _, vlan := range c.Vlan {
		if isVlansSet(vlan, c.Port) {
			log.Trace().Msg("vlan already set")
			continue
		}
		addVlanToPort(vlan, c.Port)
	}
}

func (c *CurrConfig) RemoveVlansFromPort() {
	log.Trace().Msg("RemoveVlanAsMaster")
	for _, vlan := range c.Vlan {
		if !isVlansSet(vlan, c.Port) {
			log.Trace().Msg("vlan already removed")
			continue
		}
		delVlanFromPort(vlan, c.Port)
	}
}

func addVlanToPort(vlan, port string) {
	//sudo config vlan member add 702 Ethernet125
	cmd := sudoCMD + " config vlan member add %s %s"
	_, err := shell.Commandf(cmd, vlan, port).Output()
	if err != nil {
		log.Trace().Msg("Add vlan failed: " + err.Error())
	}
}
func delVlanFromPort(vlan, port string) {
	//sudo config vlan member del 702 Ethernet125
	cmd := sudoCMD + " config vlan member del %s %s"
	_, err := shell.Commandf(cmd, vlan, port).Output()
	if err != nil {
		log.Trace().Msg("Del vlan failed: " + err.Error())
	}
}

func isVlansSet(vlan, port string) bool {
	/*
	   admin@7726-a42:~$ show vlan config | grep 702
	   Vlan702     702  Ethernet0       tagged
	   Vlan702     702  Ethernet36      tagged
	   Vlan702     702  Ethernet125     tagged
	*/
	cmd := showCMD + " vlan config | grep -v grep | grep %s | grep %s"
	out, err := shell.Commandf(cmd, vlan, port).Output()
	if err != nil {
		log.Trace().Msg("show vlan failed: " + err.Error())
		return false
	}
	if len(string(out)) == 0 {
		return false
	}
	tmp := strings.Fields(string(out))
	if len(tmp) == 0 {
		return false
	}
	return true
}

func parseVlans(arg string) []string {
	var vlanList []string
	matched, err := regexp.MatchString(`^\d+,.*`, arg)
	if err != nil {
		log.Trace().Msg("parseVlans failed")
	}
	if matched {
		vlanList = strings.Split(arg, ",")
	} else {
		vlanList = append(vlanList, arg)
	}
	return vlanList
}
