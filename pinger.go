package main

import "fmt"
import "os"
import "sync"
import ping "github.com/sparrc/go-ping"

func ping_host(host string){
  pinger, err := ping.NewPinger(host)
  if err != nil {
    fmt.Printf("Ping failed: %s\n", err)
    return
  }
  pinger.Count = 1
  pinger.Run()
  stats := pinger.Statistics()
  fmt.Printf("Sent %d\n", stats.PacketsSent)
}

func main(){
  hosts := make([]string, 0)
  for _, arg := range os.Args[1:] {
    fmt.Println(arg)
    hosts = append(hosts, arg)
  }

  var waiter sync.WaitGroup
  for _, host := range hosts {
      waiter.Add(1)
      go func(host string){
          defer waiter.Done()
          ping_host(host)
      }(host)
  }
  waiter.Wait()
}
