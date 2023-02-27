package main

import (
	"errors"
	"github.com/rs/zerolog/log"
	"net"
	"net/rpc"
	"strconv"
	"time"
)

type Listener struct {
	Sleep time.Duration
	c     *CurrConfig
}

const PORT = "58085"

func runServer(cfg *CurrConfig) error {
	addr := cfg.Local
	bind := addr + ":" + PORT
	bindaddr, err := net.ResolveTCPAddr("tcp", bind)
	if err != nil {
		log.Trace().Msg(err.Error())
		return err
	}

	inbound, err := net.ListenTCP("tcp", bindaddr)
	if err != nil {
		log.Trace().Msg(err.Error())
		return err
	}

	listener := new(Listener)
	rpc.Register(listener)
	go rpc.Accept(inbound)
	listener.c = cfg
	log.Trace().Msg("RPCSRV started " + bind)
	return nil
}

func (l *Listener) Pong(line []byte, ack *bool) (err error) {
	//user := string(line)
	//args := []string{user}
	*ack = true
	return
}

func (l *Listener) AddVlans(line []byte, ack *bool) (err error) {
	for _, vlan := range l.c.Vlan {
		if isVlansSet(vlan, l.c.Port) {
			log.Trace().Msg("vlan already exists")
			continue
		}
		addVlanToPort(vlan, l.c.Port)
	}
	*ack = true
	return
}

func (l *Listener) RemoveVlans(line []byte, ack *bool) (err error) {
	for _, vlan := range l.c.Vlan {
		if !isVlansSet(vlan, l.c.Port) {
			log.Trace().Msg("vlan already removed")
			continue
		}
		delVlanFromPort(vlan, l.c.Port)
	}
	*ack = true
	return
}

//Client side requests
func rpcPing(ip, cmd string) error {
	srvAddr := ip + ":" + PORT
	client, err := rpc.Dial("tcp", srvAddr)
	if err != nil {
		log.Trace().Msg(srvAddr + " Dial failed")
		return err
	}
	defer client.Close()

	reply := false
	var line []byte

	log.Trace().Msg(strconv.FormatBool(reply))

	line = []byte("ping")
	rCmd := "Pong"

	if cmd != "" && cmd != "ping" {
		rCmd = cmd
		line = []byte(cmd)
	}

	waitRpc := client.Go("Listener."+rCmd, line, &reply, nil)
	<-waitRpc.Done

	if waitRpc.Error != nil {
		return waitRpc.Error
	}
	log.Trace().Msg(strconv.FormatBool(reply))
	if !reply {
		return errors.New("Peer " + ip + " not available")
	}
	return nil
}
