package main

import "fmt"
import ping "github.com/sparrc/go-ping"

func main(){
  pinger, err := ping.NewPinger("www.google.com")
  fmt.Println("hello")
  if err != nil {
    panic(err)
  }
  pinger.Count = 3
  pinger.Run()
  stats := pinger.Statistics()
  fmt.Printf("Sent %d\n", stats.PacketsSent)
}
