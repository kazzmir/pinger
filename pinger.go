package main

import "fmt"
import "os"
import "sort"
import _ "net/http/pprof"
import _ "net/http"
import _ "log"
// import "math/rand"
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

func Min(x, y int) int {
    if x > y {
        return y
    }
    return x
}

type StatusSort struct {
  data []Status
}

func (s StatusSort) Len() int {
  return len(s.data)
}

func (s StatusSort) Less(i, j int) bool {
  a := s.data[i]
  b := s.data[j]

  if a.last_ping == -1 && b.last_ping > -1 {
    return false
  }

  if b.last_ping == -1 && a.last_ping > -1 {
    return true
  }

  if a.last_ping == b.last_ping {
    return a.host < b.host
  }
  return a.last_ping < b.last_ping
}

func (s StatusSort) Swap(i, j int) {
  t := s.data[i]
  s.data[i] = s.data[j]
  s.data[j] = t
}

func sort_hosts_by_ping(hosts map[string]Status, reverse bool) []string {
  keys := []Status{}
  for _, k := range hosts {
    keys = append(keys, k)
  }
  data := StatusSort{keys}
  if reverse {
    sort.Sort(sort.Reverse(data))
  } else {
    sort.Sort(data)
  }

  names := []string{}
  for _, status := range data.data {
    names = append(names, status.host)
  }
  return names
}

func sort_hosts_by_name(hosts map[string]Status, reverse bool) []string {
  keys := []string{}
  for k, _ := range hosts {
      keys = append(keys, k)
  }
  sort.Strings(keys)
  if reverse && len(keys) > 0 {
    /* FIXME: theres probably a library method for reversing an array */
    upper := len(keys) / 2
    for i := 0; i < upper; i += 1 {
      s := keys[i]
      keys[i] = keys[len(keys) - i - 1]
      keys[len(keys) - i - 1] = s
    }
  }
  return keys
}

const SortByName = 0
const SortByNameReverse = 1
const SortByPing = 2
const SortByPingReverse = 3

func sort_hosts(hosts map[string]Status, sort_type int) []string {
  switch sort_type {
    case SortByName: return sort_hosts_by_name(hosts, false)
    case SortByNameReverse: return sort_hosts_by_name(hosts, true)
    case SortByPing: return sort_hosts_by_ping(hosts, false)
    case SortByPingReverse: return sort_hosts_by_ping(hosts, true)
  }
  return nil
}

func sort_description(sort_type int) string {
  switch sort_type {
    case SortByName: return "(S)ort by name"
    case SortByNameReverse: return "(S)ort by name (reversed)"
    case SortByPing: return "(S)ort by ping time"
    case SortByPingReverse: return "(S)ort by ping time (reversed)"
  }
  return ""
}

func render(hosts map[string]Status, scroll int, sort_type int){
  background := termbox.ColorBlack
  termbox.Clear(termbox.ColorWhite, background)

  term_print(0, 0, termbox.ColorWhite, background, fmt.Sprintf("Pinger version %d.%d", VERSION_MAJOR, VERSION_MINOR))
  // term_print(30, 0, termbox.ColorYellow, background, ([]string{"1", "2", "3", "4", "5", "6", "7", "8"})[rand.Intn(8)])

  now := time.Now()
  hour := now.Hour()
  ampm := "am"
  if hour >= 12 {
      ampm = "pm"
  }
  if hour > 12 {
      hour -= 12
  }
  term_print(0, 1, termbox.ColorWhite, background, fmt.Sprintf("%d/%02d/%d %d:%02d:%02d%s", now.Year(), now.Month(), now.Day(), hour, now.Minute(), now.Second(), ampm))

  term_print(30, 1, termbox.ColorWhite, background, sort_description(sort_type))

  x := 2
  y := 2

  _, screen_height := termbox.Size()

  max_display := screen_height - y

  var status_x int = x
  keys := sort_hosts(hosts, sort_type)
  end := Min(len(keys) - scroll, max_display)

  if scroll > 0 {
      term_print(0, 2, termbox.ColorYellow, background, "^")
      term_print(0, 3, termbox.ColorYellow, background, "|")
      term_print(0, 4, termbox.ColorYellow, background, "|")
  }

  if scroll + end < len(keys) {
      term_print(0, screen_height - 3, termbox.ColorYellow, background, "|")
      term_print(0, screen_height - 2, termbox.ColorYellow, background, "|")
      term_print(0, screen_height - 1, termbox.ColorYellow, background, "V")
  }

  for _, host := range keys[scroll:scroll + end] {
      status_x = Max(status_x, len(host) + x + 3)
  }

  for _, host := range keys[scroll:scroll + end] {
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
  last_ping time.Duration
}

const ScrollUp = 0
const ScrollDown = 1
const ScrollPageUp = 2
const ScrollPageDown = 3
const Repaint = 4

func display(hosts []string){
  err := termbox.Init()
  if err != nil {
    panic(err)
  }
  defer termbox.Close()

  var state map[string]Status
  state = make(map[string]Status)
  var state_update = make(chan Status, len(hosts) * 3)

  var sort_type = SortByName

  /* Number of simaltaenous pings being sent. What is a good number? Number of cores? */
  can_ping := make(chan int)
  go func(){
      for {
          for i := 0; i < 7; i++ {
            can_ping <- 0
          }
          time.Sleep(1 * time.Second)
      }
  }()

  for _, host := range hosts {
      state[host] = Status{host, "...", false, -1}
      go func(host string){
          for {
              // sleep_time := time.Duration(1300 + 200 * rand.Intn(10)) * time.Millisecond
              <-can_ping
              stats, err := ping_host(host)
              if err != nil {
                  state_update <- Status{host, err.Error(), false, -1}
                  // sleep_time = time.Duration(rand.Intn(10) + 3) * time.Second
              } else {
                  state_update <- Status{host, stats.AvgRtt.String(), true, stats.AvgRtt}
              }
              // time.Sleep(sleep_time)
          }
      }(host)
  }

  render(state, 0, sort_type)

  second := make(chan bool)
  go func(){
      for {
          time.Sleep(1 * time.Second)
          second <- true
      }
  }()

  action := make(chan int, 100)

  go func(){
    scroll := 0
    for {
        // fmt.Printf("Wait..\n")
        refresh := false
        // fmt.Printf("Go..\n")
        all:
        for {
          select {
            case move := <-action: {
              refresh = true
                screen_width, screen_height := termbox.Size()
              _ = screen_width
              _ = screen_height

              movement := 0
              if move == ScrollDown {
                  movement = 1
              } else if move == ScrollPageDown {
                  movement = 10
              } else if move == ScrollUp {
                  movement = -1
              } else if move == ScrollPageUp {
                  movement = -10
              }

              scroll += movement
              max_up := len(hosts) - screen_height + 2
              if scroll >= max_up {
                scroll = max_up
              }
              if scroll < 0 {
                  scroll = 0
              }
              break
            }
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
          render(state, scroll, sort_type)
        } else {
          time.Sleep(1 * time.Millisecond)
        }
    }
  }()

  for {
      event := termbox.PollEvent()
      if event.Type == termbox.EventKey {
        key := event.Key
        // fmt.Println(key)
        if key == termbox.KeyArrowUp {
            action <- ScrollUp
        }
        if key == termbox.KeyArrowDown {
            action <- ScrollDown
        }
        if key == termbox.KeyPgup {
            action <- ScrollPageUp
        }
        if key == termbox.KeyPgdn {
            action <- ScrollPageDown
        }
        if event.Ch == 's' {
          action <- Repaint
          switch sort_type {
            case SortByName: {
              sort_type = SortByNameReverse
            }
            case SortByNameReverse: {
              sort_type = SortByPing
            }
            case SortByPing: {
              sort_type = SortByPingReverse
            }
            case SortByPingReverse: {
              sort_type = SortByName
            }
          }
        }
        if key == termbox.KeyEsc || event.Ch == 'q' {
            break
        }
      }
  }
}

func is_ip_with_netmask(host string) bool {
    matched, _ := regexp.MatchString("\\d+\\.\\d+\\.\\d+\\.\\d+\\/\\d+", host)
    return matched
}

/*
 * /32 => [255, 255, 255, 255]
 * /27 => [255, 255, 255, 224]
 * /24 => [255, 255, 255, 0]
 */
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
    out := make(chan int)
    max := 255 - bits
    go func(){
        for i := 0; i < max + 1 && i + number <= 255; i++ {
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
    out = append(out, scanner.Text())
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
    if arg == "-h" {
      if i + 1 < len(os.Args) {
        i += 1
        for _, host := range read_file(os.Args[i]) {
          hosts = append(hosts, process_host(host)...)
        }
      }
    } else {
      hosts = append(hosts, process_host(arg)...)
    }
  }

  display(hosts)

  _ = fmt.Println
}
