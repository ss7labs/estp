package main

import (
	"errors"
	"github.com/go-ping/ping"
	"github.com/rs/zerolog/log"
	"strconv"
	"time"
)

func checkHostAvail(host string) error {
	var rcvd int
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return err
	}
	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Timeout = time.Second * 1
	pinger.OnFinish = func(stats *ping.Statistics) {
		/*
			fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
			fmt.Printf("%d packets transmitted, %d packets received, %d duplicates, %v%% packet loss\n",
				stats.PacketsSent, stats.PacketsRecv, stats.PacketsRecvDuplicates, stats.PacketLoss)
			fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
				stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
		*/
		log.Trace().Msg(stats.Addr + " packets received " + strconv.Itoa(stats.PacketsRecv))
		rcvd = stats.PacketsRecv
	}

	err = pinger.Run()
	if err != nil {
		log.Trace().Msg("Pinger " + host + " " + err.Error())
		return err
	}
	//fmt.Println("PING ", err, pinger.IPAddr(),rcvd)
	if rcvd == 0 {
		err = errors.New("Host not available")
		return err
	}
	return nil
}
