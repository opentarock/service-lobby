package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"

	"github.com/opentarock/service-api/go/client"
	nservice "github.com/opentarock/service-api/go/service"

	"github.com/opentarock/service-api/go/proto_lobby"
	"github.com/opentarock/service-lobby/service"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	// profiliing related flag
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	lobbyService := nservice.NewRepService("tcp://*:7001")

	notifyClient := client.NewNotifyClientNanomsg()
	notifyClient.Connect("tcp://localhost:8001")
	defer notifyClient.Close()

	handlers := service.NewLobbyServiceHandlers(notifyClient)
	lobbyService.AddHandler(proto_lobby.CreateRoomRequestMessage, handlers.CreateRoomHandler())
	lobbyService.AddHandler(proto_lobby.JoinRoomRequestMessage, handlers.JoinRoomHandler())
	lobbyService.AddHandler(proto_lobby.LeaveRoomRequestMessage, handlers.LeaveRoomHandler())
	lobbyService.AddHandler(proto_lobby.ListRoomsRequestMessage, handlers.ListRoomsHandler())
	lobbyService.AddHandler(proto_lobby.RoomInfoRequestMessage, handlers.RoomInfoHandler())

	err := lobbyService.Start()
	if err != nil {
		log.Fatalf("Error starting lobby service: %s", err)
	}
	defer lobbyService.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	sig := <-c
	log.Printf("Interrupted by %s", sig)
}
