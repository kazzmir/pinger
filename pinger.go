package main

import "fmt"
import "os"
import "sort"
import _ "net/http/pprof"
import _ "net/http"
import _ "log"
// import "runtime/debug"
import "strconv"
import "bufio"
import "github.com/nsf/termbox-go"
import "sync"
import "regexp"
import "errors"
import "time"
import "github.com/sparrc/go-ping"

const VERSION_MAJOR = 0
const VERSION_MINOR = 1

func ping_host(host string) (ping.Statistics, error) {
  pinger, err := ping.NewPinger(host)
  if err != nil {
    // fmt.Printf("Ping failed: %s\n", err)
    s := ping.Statistics{}
    return s, err
  }
  pinger.Count = 1
  pinger.Timeout = 2 * time.Second
  pinger.Run()
  stats := pinger.Statistics()
  if stats.AvgRtt == 0 {
      return ping.Statistics{}, errors.New("timeout")
  }
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

func render(hosts map[string]Status){
  background := termbox.ColorBlack
  termbox.Clear(termbox.ColorWhite, background)

  term_print(0, 0, termbox.ColorWhite, termbox.ColorBlack, fmt.Sprintf("Pinger version %d.%d", VERSION_MAJOR, VERSION_MINOR))

  now := time.Now()
  hour := now.Hour()
  ampm := "am"
  if hour > 12 {
      hour -= 12
      ampm = "pm"
  }
  term_print(1, 1, termbox.ColorWhite, termbox.ColorBlack, fmt.Sprintf("%d/%02d/%d %d:%02d:%02d%s", now.Year(), now.Month(), now.Day(), hour, now.Minute(), now.Second(), ampm))

  x := 1
  y := 2
  var status_x int = 1
  keys := []string{}
  for k, _ := range hosts {
      keys = append(keys, k)
      status_x = Max(status_x, len(k) + 2)
  }
  sort.Strings(keys)
  for _, host := range keys {
      status := hosts[host]
      foreground := termbox.ColorGreen
      if ! status.ok {
          foreground = termbox.ColorRed
      }
      term_print(x, y, foreground, background, host)
      term_print(status_x, y, foreground, background, status.message)
      y += 1
  }
  termbox.Flush()
}

func f(){
    var waiter sync.WaitGroup
    waiter.Done()
}

type Status struct {
  host string
  message string
  ok bool
}

func display(hosts []string){
  err := termbox.Init()
  if err != nil {
    panic(err)
  }
  defer termbox.Close()

  var state map[string]Status
  state = make(map[string]Status)
  var state_update = make(chan Status, len(hosts) * 3)

  for _, host := range hosts {
      state[host] = Status{host, "...", false}
      go func(host string){
          for {
              stats, err := ping_host(host)
              if err != nil {
                  state_update <- Status{host, err.Error(), false}
              } else {
                  state_update <- Status{host, stats.AvgRtt.String(), true}
              }
              time.Sleep(1 * time.Second)
          }
      }(host)
  }

  render(state)

  second := make(chan bool)
  go func(){
      for {
          time.Sleep(1 * time.Second)
          second <- true
      }
  }()

  go func(){
    for {
        // fmt.Printf("Wait..\n")
        refresh := false
        time.Sleep(200 * time.Millisecond)
        // fmt.Printf("Go..\n")
        all:
        for {
          select {
            case update := <-state_update: {
              refresh = true
              state[update.host] = update
              break
            }
            case <-second: {
              refresh = true
              break
            }
            default: {
              // fmt.Printf("Nothing\n")
              break all
            }
          }
        }
        if refresh {
          // fmt.Printf("Render..\n")
          render(state)
        }
    }
  }()

  termbox.PollEvent()
}

func is_ip_with_netmask(host string) bool {
    matched, _ := regexp.MatchString("\\d+\\.\\d+\\.\\d+\\.\\d+\\/\\d+", host)
    return matched
}

func get_subnet_netmask(subnet uint) []int {
    m32 := 0xffffffff
    bits := m32 & (m32 >> subnet)
    n := m32 ^ bits

    b1 := (n >> 24) & 0xff
    b2 := (n >> 16) & 0xff
    b3 := (n >> 8) & 0xff
    b4 := (n >> 0) & 0xff
    return []int{b1, b2, b3, b4}
}

func make_part(number, bits int) chan int {
    /*
    out := make(chan int, 255)
    max := 255 - bits
    for i := 0; i < max + 1; i++ {
        out <- number + i
    }
    close(out)
    return out
    */

    out := make(chan int)
    max := 255 - bits
    go func(){
        for i := 0; i < max + 1; i++ {
            out <- number + i
        }
        close(out)
    }()

    return out
}

func generate_ips(b1, b2, b3, b4, subnet int) []string {
    netmask := get_subnet_netmask(uint(subnet))
    out := make([]string, 0)

    /*
    go func(){
        for {
            time.Sleep(1 * time.Second)
            debug.PrintStack()
        }
    }()
    */

    // fmt.Println("%d.%d.%d.%d/%d\n", b1, b2, b3, b4, subnet)
    for a := range make_part(b1, netmask[0]) {
        for b := range make_part(b2, netmask[1]) {
            for c := range make_part(b3, netmask[2]) {
                for d := range make_part(b4, netmask[3]) {
                    // fmt.Printf("%d.%d.%d.%d\n", a, b, c, d)
                    if d != 255 {
                        out = append(out, fmt.Sprintf("%d.%d.%d.%d", a, b, c, d))
                    }
                }
            }
        }
    }

    return out
}

/*
 * hostname => [hostname]
 * ip => [ip]
 * ip/bits => [ip, ip+1, ip+2, ...]
 *
 * meaning 172.16.0.0/24 will return all addresses from 172.16.0.0-172.16.0.254
 * the broadcast ip, .255, will be left off
 */
func process_host(host string) []string {
    if is_ip_with_netmask(host) {
        // fmt.Printf("%s is a netmask\n", host)
        ip := regexp.MustCompile("(\\d+)\\.(\\d+)\\.(\\d+)\\.(\\d+)\\/(\\d+)")
        parts := ip.FindStringSubmatch(host)
        b1, _ := strconv.Atoi(parts[1])
        b2, _ := strconv.Atoi(parts[2])
        b3, _ := strconv.Atoi(parts[3])
        b4, _ := strconv.Atoi(parts[4])
        netmask, _ := strconv.Atoi(parts[5])
        return generate_ips(b1, b2, b3, b4, netmask)
    }

    return []string{host}
}

func read_file(path string) []string {
  out := make([]string, 0)

  file, err := os.Open(path)
  if err != nil {
    panic(err)
  }
  defer file.Close()

  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    out = append(out, process_host(scanner.Text())...)
  }

  return out
}

func main(){

    /*
  go func(){
    log.Println(http.ListenAndServe("localhost:6060", nil))
  }()
  */

  // fmt.Println(get_subnet_netmask(24))
  // fmt.Println(generate_ips(172, 16, 0, 0, 30))
  hosts := make([]string, 0)
  for i := 1; i < len(os.Args); i++ {
    arg := os.Args[i]
    fmt.Println(arg)
    if arg == "-h" {
      if i + 1 < len(os.Args) {
        i += 1
        hosts = append(hosts, read_file(os.Args[i])...)
      }
    } else {
      hosts = append(hosts, process_host(arg)...)
    }
    fmt.Println(arg)
  }

  fmt.Println("go\n")
  display(hosts)

  _ = fmt.Println
}
