package main

import (
	"github.com/rs/zerolog/log"
	"net"
	"net/rpc"
  "time"
  "strconv"
  "errors"
)

type Listener struct {
	Sleep time.Duration
}

const PORT = "58085"

func runServer(addr string) error {
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
	log.Trace().Msg("RPCSRV started "+bind)
  return nil
}

func (l *Listener) Pong(line []byte, ack *bool) (err error) {
	//user := string(line)
	//args := []string{user}
 *ack = true
 return
}

func rpcPing(ip string) error {
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

  waitRpc := client.Go("Listener.Pong", line, &reply, nil)
	<-waitRpc.Done

  if waitRpc.Error != nil {
		return waitRpc.Error
	}
	log.Trace().Msg(strconv.FormatBool(reply))
  if !reply {
    return errors.New("Peer "+ip+" not available")
  }
  return nil
}

