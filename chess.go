package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/desertbit/glue"
	"github.com/gobuffalo/packr"
	"github.com/spf13/pflag"
)

//go:generate packr -v -z
func main() {
	addr := pflag.StringP("addr", "a", ":8095", "address to listen on")
	assets := pflag.StringP("assets", "s", "", "the directory where the assets are stored (default: embedded)")
	help := pflag.BoolP("help", "h", false, "show this message")
	pflag.Parse()

	if *help || pflag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "Usage: chess [OPTIONS]\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if *assets != "" {
		http.Handle("/", http.FileServer(http.Dir(*assets)))
	} else {
		http.Handle("/", http.FileServer(packr.NewBox("./public")))
	}

	socket := glue.NewServer(glue.Options{
		HTTPListenAddress: *addr,
	})

	gameSockets := map[string]string{}
	gameBoards := map[string]string{
		"default": "",
	}

	sendToGame := func(id, data string) {
		for _, s := range socket.Sockets() {
			if !s.IsClosed() {
				for sid, gid := range gameSockets {
					if sid == s.ID() && gid == id {
						s.Write(data)
					}
				}
			}
		}
	}

	socket.OnNewSocket(func(s *glue.Socket) {
		fmt.Println(s.ID(), "recieved new connection")

		var gid string

		s.OnRead(func(data string) {
			fmt.Println("socket:", s.ID(), "game:", gameSockets[s.ID()], "data:", data)
			switch {
			case strings.HasPrefix(data, "connect"):
				gid = s.ID()[:5]
				if spl := strings.Split(data, ":"); len(spl) >= 2 {
					gid = spl[1]
				}
				fmt.Println(s.ID(), "recieved connect", gid)

				gameSockets[s.ID()] = gid

				fmt.Println(s.ID(), "--> sending id", gid)
				s.Write("id:" + gid)

				if b, ok := gameBoards[gid]; ok && b != "" {
					fmt.Println(s.ID(), "--> found board for existing game", gid, ", sending update")
					s.Write(b)
				} else {
					fmt.Println(s.ID(), "--> no existing game for gid", gid, ", sending reset")
					sendToGame(gid, "reset")
				}
			case data == "reset":
				fmt.Println(s.ID(), "recieved reset for gid", gid)
				if gid == "" {
					fmt.Println(s.ID(), "--> no gid set yet, skipping")
					s.Write("temperror:No game ID set")
					return
				}
				gameBoards[gid] = ""
				sendToGame(gid, "reset")
			default:
				fmt.Println(s.ID(), "recieved board update for gid", gid)
				if gid == "" {
					fmt.Println(s.ID(), "--> no gid set yet, skipping")
					s.Write("temperror:No game ID set")
					return
				}
				fmt.Println(s.ID(), "--> sending board update to other sockets for gid", gid)
				gameBoards[gid] = data
				sendToGame(gid, data)
			}
		})
	})

	defer socket.Release()

	fmt.Printf("Listening on %s\n", *addr)
	panic(socket.Run())
}
