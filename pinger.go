package main

import "fmt"
import "os"
import "github.com/nsf/termbox-go"
import "sync"
import "github.com/sparrc/go-ping"

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

func term_print(x, y int, fg, bg termbox.Attribute, message string){
  for _, c := range message {
    termbox.SetCell(x, y, c, fg, bg)
    x += 1
  }
}

func display(hosts []string){
  err := termbox.Init()
  if err != nil {
    panic(err)
  }
  defer termbox.Close()
  termbox.Clear(termbox.ColorBlue, termbox.ColorBlue)

  x := 1
  y := 1
  for _, host := range hosts {
      term_print(x, y, termbox.ColorRed, termbox.ColorWhite, host)
      y += 1
  }
  termbox.Flush()

  termbox.PollEvent()
}

func main(){
  hosts := make([]string, 0)
  for _, arg := range os.Args[1:] {
    fmt.Println(arg)
    hosts = append(hosts, arg)
  }

  display(hosts)

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
