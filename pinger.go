package main

import "fmt"
import "os"
import "sort"
import "github.com/nsf/termbox-go"
import "sync"
import "time"
import "github.com/sparrc/go-ping"

func ping_host(host string) (ping.Statistics, error) {
  pinger, err := ping.NewPinger(host)
  if err != nil {
    // fmt.Printf("Ping failed: %s\n", err)
    s := ping.Statistics{}
    return s, err
  }
  pinger.Count = 1
  pinger.Run()
  stats := pinger.Statistics()
  return *stats, nil
  // fmt.Printf("Sent %d\n", stats.PacketsSent)
}

func term_print(x, y int, fg, bg termbox.Attribute, message string){
  for _, c := range message {
    termbox.SetCell(x, y, c, fg, bg)
    x += 1
  }
}

func Max(x, y int) int {
    if x > y {
        return x
    }
    return y
}

func render(hosts map[string]string){
  background := termbox.ColorBlack
  termbox.Clear(background, termbox.ColorWhite)

  x := 1
  y := 1
  var status_x int = 1
  keys := []string{}
  for k, _ := range hosts {
      keys = append(keys, k)
      status_x = Max(status_x, len(k) + 2)
  }
  sort.Strings(keys)
  for _, host := range keys {
      status := hosts[host]
      term_print(x, y, termbox.ColorRed, termbox.ColorWhite, host)
      term_print(status_x, y, termbox.ColorRed, termbox.ColorWhite, status)
      y += 1
  }
  termbox.Flush()
}

func f(){
    var waiter sync.WaitGroup
    waiter.Done()
}

type Update struct {
  host string
  message string
}

func display(hosts []string){
  err := termbox.Init()
  if err != nil {
    panic(err)
  }
  defer termbox.Close()

  var state map[string]string
  state = make(map[string]string)
  var state_update = make(chan Update, len(hosts) * 3)

  for _, host := range hosts {
      state[host] = "..."
      go func(host string){
          for {
              stats, err := ping_host(host)
              if err != nil {
                  state_update <- Update{host, err.Error()}
              } else {
                  state_update <- Update{host, stats.AvgRtt.String()}
              }
              time.Sleep(1 * time.Second)
          }
      }(host)
  }

  render(state)

  go func(){
    for {
        // fmt.Printf("Wait..\n")
        time.Sleep(1 * time.Second)
        // fmt.Printf("Go..\n")
        all:
        for {
          select {
            case update := <-state_update: {
              state[update.host] = update.message
              break
            }
            default: {
              // fmt.Printf("Nothing\n")
              break all
            }
          }
        }
        // fmt.Printf("Render..\n")
        render(state)
    }
  }()

  termbox.PollEvent()
}

func main(){
  hosts := make([]string, 0)
  for _, arg := range os.Args[1:] {
    fmt.Println(arg)
    hosts = append(hosts, arg)
  }

  display(hosts)
}
