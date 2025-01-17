package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Abort bool
	mtx   sync.Mutex
)

// Check for update
func CheckUpdate(currentTag string) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	respJson, status, err := request("https://api.github.com/repos/baconator696/Recu-Download/releases/latest", 2, nil, nil, "GET")
	if err != nil {
		return
	} else if status != 200 {
		return fmt.Errorf("status: %d, %s", status, string(respJson))
	}
	var resp any
	err = json.Unmarshal(respJson, &resp)
	if err != nil {
		return
	}
	if resp.(map[string]any)["prerelease"].(bool) {
		return
	}
	newTag := resp.(map[string]any)["tag_name"].(string)
	newTag = strings.ReplaceAll(newTag, "v", "")
	newNums := strings.Split(newTag, ".")
	currentTag = strings.ReplaceAll(currentTag, "v", "")
	currentNums := strings.Split(currentTag, ".")
	for i, v := range newNums {
		current, err := strconv.Atoi(currentNums[i])
		if err != nil {
			continue
		}
		new, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		if new > current {
			fmt.Printf("New Update Available: v%s\n", newTag)
			fmt.Printf("%s\n%s\n", resp.(map[string]any)["html_url"].(string), ANSIColor(resp.(map[string]any)["body"].(string), 2))
			return nil
		}
	}
	return nil
}

// Returns the raw data from the URL
func request(url string, timeout int, header map[string]string, body []byte, Type string) ([]byte, int, error) {
	req, err := http.NewRequest(Type, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, 0, fmt.Errorf("http.NewRequest:%v", err)
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	data, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("client.Do:%v", err)
	}
	defer data.Body.Close()
	databytes, err := io.ReadAll(data.Body)
	if err != nil {
		return nil, data.StatusCode, fmt.Errorf("io.ReadAll:%v", err)
	}
	return databytes, data.StatusCode, nil
}

// Parses executatables arguments to prevent runtime errors
func Argparser(n int) string {
	if len(os.Args) > n {
		return os.Args[n]
	}
	return ""
}

// ANSI Color
func ANSIColor(str any, mod int, color ...int) (final string) {
	var x, r, g, b int
	var rgb bool
	if len(color) == 1 {
		x = color[0]
	} else if len(color) == 3 {
		rgb = true
		r = color[0]
		g = color[1]
		b = color[2]
	}
	var res int
	switch {
	case mod == 1:
		res = 22
	case mod == 21:
		res = 24
	case mod >= 2 && mod <= 9:
		res = mod + 20
	case (mod >= 30 && mod <= 38) || (mod >= 90 && mod <= 97):
		res = 39
	case (mod >= 40 && mod <= 48) || (mod >= 100 && mod <= 107):
		res = 49
	}
	if mod == 38 || mod == 48 {
		if rgb {
			final = fmt.Sprintf("\033[%d;2;%d;%d;%dm%v\033[%dm", mod, r, g, b, str, res)
		} else {
			final = fmt.Sprintf("\033[%d;5;%dm%v\033[%dm", mod, x, str, res)
		}
	} else {
		final = fmt.Sprintf("\033[%dm%v\033[%dm", mod, str, res)
	}
	return
}

// Looks for the first occurence of start and end and returns the string in between
func searchString(str string, start string, end string) (string, error) {
	if len(str) <= len(start)+len(end) {
		return "", fmt.Errorf("search term longer than the given string")
	}
	index1 := strings.Index(str, start)
	index2 := strings.Index(str[index1+len(start):], end)
	if index1 == -1 || index2 == -1 {
		return "", fmt.Errorf("could not find {%v} and/or {%v} in {%v}", start, end, ANSIColor(str, 2))
	}
	return str[index1+len(start) : index1+len(start)+index2], nil
}

// String Shorten
func shortenString(str any, ln int) string {
	if ln < 0 {
		ln = 0
	}
	switch i := str.(type) {
	case string:
		if len(i) > ln {
			return i[:ln]
		} else {
			return i
		}
	case error:
		if len(i.Error()) > ln {
			return i.Error()[:ln]
		} else {
			return i.Error()
		}
	default:
		return fmt.Sprintf("Type:%v", i)
	}
}

// convert timestamps into percent
func percentPrase(times []any) []float64 {
	var start, end float64
	var secs [3]int
	for i, w := range times {
		v, ok := w.(string)
		if !ok {
			fmt.Fprintf(os.Stderr, "timestamps is in wrong format: %v\n", times)
			return nil
		}
		time := strings.Split(v, ":")
		cons := 1
		for j := len(time) - 1; j >= 0; j-- {
			w, err := strconv.Atoi(time[j])
			if err != nil {
				fmt.Fprintf(os.Stderr, "timestamps is in wrong format: %v\n", times)
				return nil
			}
			secs[i] += w * cons
			cons *= 60
		}
	}
	start = float64(secs[0]) / float64(secs[2]) * 100
	end = float64(secs[1]) / float64(secs[2]) * 100
	return []float64{start, end}
}

// Defines the Average Buffer
type AvgBuffer struct {
	data []float64
	pos  int
	size int
}

// Returns the Average of the buffer
func (buff AvgBuffer) average() (avg float64) {
	for _, value := range buff.data {
		avg += value
	}
	avg /= float64(len(buff.data))
	return
}

// Adds a number to the average buffer
func (buff *AvgBuffer) add(add float64) {
	if buff.size <= 0 {
		buff.size = 25
	}
	if buff.pos < 0 || buff.pos >= buff.size {
		buff.pos = 0
	}
	for buff.pos >= len(buff.data) {
		buff.data = append(buff.data, add)
	}
	buff.data[buff.pos] = add
	buff.pos++
}

// Converts int in Seconds to a formated string
func formatMinutes(num float64) string {
	var unit string
	switch true {
	case num < 1:
		num *= 60
		unit = "secs"
	case num > 1440:
		num /= 1440
		unit = "days"
	case num > 60:
		num /= 60
		unit = "hours"
	default:
		unit = "mins"
	}
	return fmt.Sprintf("%.1f %s", num, unit)
}

// Converts Number of Bytes per second to a formated string
func formatBytesPerSecond(num float64) string {
	var unit string
	switch true {
	case num < 1000:
		unit = "B/s"
	case num >= 1000000:
		num /= 1000000
		unit = "MB/s"
	case num >= 1000:
		num /= 1000
		unit = "KB/s"
	}
	return fmt.Sprintf("%.1f %s", num, unit)
}

// Returns default templet
func TempletJSON() Templet {
	var jsonTemplet Templet
	jsonTemplet.Header = map[string]string{
		"Cookie":     "",
		"User-Agent": "",
	}
	jsonTemplet.Urls = []any{""}
	return jsonTemplet
}

// Saves Json
func SaveJson(config Templet) (err error) {
	var jsonData []byte
	jsonData, err = json.MarshalIndent(struct {
		Urls   []any             `json:"urls"`
		Header map[string]string `json:"header"`
	}{
		Urls:   config.Urls,
		Header: config.Header,
	}, "", "\t")
	if err != nil {
		return fmt.Errorf("error: Parsing Json%v", err)
	}
	jsonLocation := "config.json"
	if Argparser(1) != "" {
		jsonLocation = Argparser(1)
	}
	err = os.WriteFile(jsonLocation, jsonData, 0666)
	if err != nil {
		err = fmt.Errorf("error: Saving Json:%v", err)
		return
	}
	return
}

// Defines the JSON used
type Templet struct {
	Urls     []any             `json:"urls"`
	Header   map[string]string `json:"header"`
	Num      int               `json:"num"`      // Deprecated
	Duration []float64         `json:"duration"` // Deprecated
}

// Gets Playlist
func (config Templet) GetPlaylist(urlAny any) ([]byte, string) {
	defer func() {
		r := recover()
		if r != nil {
			fmt.Fprintf(os.Stderr, "urls are in wrong format, error: %v\n", r)
		}
	}()
	var url string
	switch t := urlAny.(type) {
	case string:
		url = t
	case []any:
		if len(t) > 0 {
			url = t[0].(string)
		} else {
			panic("no url")
		}
	default:
		panic("url is incorrect type")
	}
	data, filename, status := recurbateParser(url, config.Header)
	switch status {
	case "cloudflare":
		fmt.Fprintf(os.Stderr, "Cloudflare Blocked: Failed on url: %v\n", url)
	case "cookie":
		fmt.Fprintf(os.Stderr, "Please Log in: Failed on url: %v\n", url)
	case "wait":
		fmt.Fprintf(os.Stderr, "Daily View Used: Failed on url: %v\n", url)
	case "panic":
		fmt.Fprintf(os.Stderr, "Panic: Failed on url: %v\n", url)
	case "done":
		return data, filename
	}
	return nil, ""
}

// Saves video to working directory
func (config *Templet) GetVideo(playlist []byte, filename string, index int) (fail int) {
	defer func() {
		r := recover()
		if r != nil {
			fmt.Fprintf(os.Stderr, "urls are in wrong format, error: %v\n", r)
			fail = 1
		}
	}()
	var url string
	var duration []float64 = config.Duration
	var num int = config.Num * -1
	switch t := config.Urls[index].(type) {
	case string:
		url = t
	case []any:
		switch len(t) {
		case 1:
			url = t[0].(string)
		case 2:
			url = t[0].(string)
			num = int(t[1].(float64))
		case 4:
			url = t[0].(string)
			duration = percentPrase(t[1:])
		case 5:
			url = t[0].(string)
			duration = percentPrase(t[1:4])
			num = int(t[4].(float64))
		default:
			panic("incorrect length of url array")
		}
	default:
		panic("url is incorrect type")
	}
	if duration == nil {
		duration = []float64{0, 100}
	}
	num = num * -1
	fail = muxPlaylist(playlist, filename, formatedHeader(config.Header, "", 0), num, duration)
	if fail == 0 {
		fmt.Printf("Completed: %v:%v\n", filename, url)
		return
	}
	fmt.Fprintf(os.Stderr, "Download Failed at line: %v\n", fail)
	switch t := config.Urls[index].(type) {
	case string:
		config.Urls[index] = []any{t, fail}
	case []any:
		switch len(t) {
		case 1:
			t = append(t, fail)
			config.Urls[index] = t
		case 2:
			t[1] = fail
			config.Urls[index] = t
		case 4:
			t = append(t, fail)
			config.Urls[index] = t
		case 5:
			t[4] = fail
			config.Urls[index] = t
		}
	}
	mtx.Lock()
	err := SaveJson(*config)
	if err != nil {
		fmt.Println(err)
	}
	mtx.Unlock()
	return
}

// Parse Urls from HTML
func (config Templet) ParseHtml(url string) (err error) {
	fmt.Println("Downloading HTML")
	resp, code, err := request(url, 10, formatedHeader(config.Header, "", 1), nil, "GET")
	if code != 200 || err != nil {
		if err == nil {
			err = fmt.Errorf("response: %s, status code: %d, cloudflare blocked", ANSIColor(string(resp), 2), code)
		}
		return
	}
	fmt.Println("Searching for Links")
	urlSplit := strings.Split(url, "/")
	name := urlSplit[4]
	prefix := strings.Join(urlSplit[:3], "/") + fmt.Sprintf("/%s/video/", name)
	suffix := "/play"
	var urls []any
	lines := strings.Split(string(resp), "\n")
	for _, v := range lines {
		code, err := searchString(v, fmt.Sprintf(`href="/%s/video/`, name), `/play"`)
		if err != nil {
			continue
		}
		urls = append(urls, prefix+code+suffix)
	}
	config.Urls = urls
	err = SaveJson(config)
	return
}

// Takes recurbate video URL and returns playlist raw data and returns file name {indexdata, filename, "done"}
func recurbateParser(url string, header map[string]string) ([]byte, string, string) {
	downloadLoop := func(url string, timeout int, header map[string]string) (data []byte, err error) {
		retry := 0
		for {
			var status int
			data, status, err = request(url, timeout, header, nil, "GET")
			if err == nil && status == 200 {
				break
			}
			fmt.Printf("Failed Retrying...\033[18D")
			if retry > 5 {
				if err == nil {
					err = fmt.Errorf("%s, status code: %d", ANSIColor(string(data), 2), status)
				}
				return
			}
			retry++
			timeout += 30
			time.Sleep(time.Millisecond * 100)
		}
		return
	}
	fmt.Printf("\rDownloading HTML: ")
	htmldata, err := downloadLoop(url, 10, formatedHeader(header, "", 1))
	if err != nil {
		fmt.Println(err)
		return nil, "", "cloudflare"
	}
	fmt.Printf("\r\033[2KDownloading HTML: Complete\n")
	token, err := searchString(string(htmldata), `data-token="`, `"`)
	if err != nil {
		fmt.Println(err)
		return nil, "", "panic"
	}
	id, err := searchString(string(htmldata)[strings.Index(string(htmldata), token):], `data-video-id="`, `"`)
	if err != nil {
		fmt.Println(err)
		return nil, "", "panic"
	}
	url = strings.Join(strings.Split(url, "/")[:3], "/") + "/api/video/" + id + "?token=" + token
	fmt.Printf("\rGetting Link to Playlist: ")
	apidata, err := downloadLoop(url, 10, formatedHeader(header, url, 2))
	if err != nil {
		fmt.Println(err)
		return nil, "", "panic"
	}
	fmt.Printf("\r\033[2KGetting Link to Playlist: Complete\n")
	switch string(apidata) {
	case "shall_subscribe":
		return nil, "", "wait"
	case "shall_signin":
		return nil, "", "cookie"
	case "wrong_token":
		fmt.Println("error: wrong token")
		return nil, "", "panic"
	}
	url, err = searchString(string(apidata), `<source src="`, `"`)
	if err != nil {
		fmt.Println(err)
		return nil, "", "panic"
	}
	url = strings.ReplaceAll(url, "amp;", "")
	fmt.Printf("\rDownloading Playlist: ")
	indexdata, err := downloadLoop(url, 10, formatedHeader(header, "", 0))
	if err != nil {
		fmt.Println(err)
		return nil, "", "panic"
	}
	fmt.Printf("\r\033[2KDownloading Playlist: Complete\n")
	filename := strings.Join(strings.Split(url, "/")[4:6], "_")
	filename = "CB_" + strings.ReplaceAll(filename, ",", "_")
	for i := 2015; i < 2050; i++ {
		year := fmt.Sprintf("%d", i)
		if strings.Contains(filename, year) {
			filename = strings.Replace(filename, year, year[2:], 1)
			break
		}
	}
	prefix := url[:strings.LastIndex(url, "/")+1]
	playlistString := string(indexdata)
	if !strings.Contains(playlistString, prefix) {
		playlistLines := strings.Split(playlistString, "\n")
		modifiedPlaylist := make([]string, len(playlistLines))
		for i, line := range playlistLines {
			if len(line) > 0 {
				if line[0] == '#' {
					modifiedPlaylist[i] = line
				} else {
					modifiedPlaylist[i] = prefix + line
				}
			}
		}
		indexdata = []byte(strings.Join(modifiedPlaylist, "\n"))
	}
	return indexdata, filename, "done"
}

// Muxes the transport streams and saves it to a file
func muxPlaylist(playlist []byte, filename string, header map[string]string, num int, duration []float64) int {
	var data []byte
	var err error
	var file *os.File
	var avgdur, avgsize AvgBuffer
	var restart int
	indexlist := strings.Split(string(playlist), "\n")
	length := len(indexlist)
	if num == 0 || num > length/2 {
		num = 1
	}
	if num%2 == 0 {
		num--
	}
	if num < 0 {
		restart = num * -1
		num = 1
	}
	if duration[0] > 100 || duration[1] <= duration[0] {
		return 0
	}
	if duration[0] < 0 {
		duration[0] = 0
	}
	if duration[1] > 100 {
		duration[1] = 100
	}
	step := int(float64(length) * duration[0] / 100)
	if restart > 0 {
		step = restart
		file, err = os.OpenFile(filename+".ts", os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "oringal file not found, creating new one: %v", err)
		}
	}
	if file == nil {
		_, err = os.Stat(filename + ".ts")
		if err == nil {
			for i := 1; i > 0; i++ {
				new := fmt.Sprintf("%s(%d)", filename, i)
				_, err := os.Stat(new + ".ts")
				if err != nil {
					filename = new
					break
				}
			}
		}
		file, err = os.OpenFile(filename+".ts", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can not create file: %v", err)
			return step
		}
	}
	defer file.Close()
	for step < int(float64(length)*duration[1]/100) {
		if len(indexlist[step]) > 0 && indexlist[step][0] != '#' {
			timeout := 10
			retry := 0
			maxRetrys := 5
			start := time.Now()
			for {
				var status int
				data, status, err = request(indexlist[step], timeout, header, nil, "GET")
				if err == nil && status == 200 {
					break
				}
				if status == 429 {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				if status == 410 {
					fmt.Fprintln(os.Stderr, "\nDownload Expired")
					retry = maxRetrys
				}
				retry++
				if err == nil {
					err = fmt.Errorf("status Code: %d, %s ", status, string(data))
				} else {
					timeout += 30
				}
				if retry > maxRetrys {
					fmt.Println()
					fmt.Fprintf(os.Stderr, "Error: %v\n", ANSIColor(err, 2))
					fmt.Fprintf(os.Stderr, "Failed at %.2f%%\n", float32(step)/float32(length)*100)
					return step
				}
				fmt.Fprintf(os.Stderr, "\n\033[2A\033[2KError: %v, Retrying...\n", ANSIColor(shortenString(err, 40), 2))
				time.Sleep(100 * time.Millisecond)
			}
			dur := time.Since(start).Minutes()
			_, err = file.Write(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "can not write file: %v", err)
				return step
			}
			// Calculate User Interface Timings
			avgsize.add(float64(len(data)))
			avgdur.add(dur)
			getavgdur := avgdur.average()
			speed := avgsize.average() / (getavgdur * 60)
			eta := getavgdur * ((float64(length) * duration[1] / 100) - float64(step)) / 2
			percent := float64(step) / float64(length) * 100
			fmt.Printf("\n\033[A\033[2KDownloading: %s\tRemaining: %s\t%s", ANSIColor(fmt.Sprintf("%.1f%%", percent), 33), formatMinutes(eta), formatBytesPerSecond(speed))
			// //
			if num > 10 {
				step += int(math.Ceil(float64(length) / float64(num)))
			} else {
				step += num
			}
		}
		step++
		if Abort {
			fmt.Println("\naborting...")
			return step
		}
	}
	fmt.Println()
	return 0
}

// Return Formated Headers, url needed only if i is 2
func formatedHeader(refHeader map[string]string, videoUrl string, i int) (header map[string]string) {
	header = make(map[string]string)
	for k, v := range refHeader {
		header[k] = v
	}
	header["Accept"] = "*/*"
	header["Accept-Language"] = "en-US,en;q=0.9"
	header["Origin"] = "https://recu.me"
	header["Priority"] = "u=1, i"
	header["Sec-Ch-Ua"] = `"Not/A)Brand";v="8", "Chromium";v="126"`
	header["Sec-Ch-Ua-Full-Version-List"] = `"Not/A)Brand";v="8.0.0.0", "Chromium";v="126.0.6478.62"`
	header["Sec-Ch-Ua-Mobile"] = "?0"
	header["Sec-Ch-Ua-Platform"] = `"Windows"`
	header["Sec-Fetch-Dest"] = "empty"
	header["Sec-Fetch-Mode"] = "cors"
	header["Sec-Ch-Ua-Arch"] = `"x86"`
	header["Sec-Ch-Ua-Bitness"] = `"64"`
	header["Sec-Ch-Ua-Full-Version"] = `"126.0.2592.61"`
	header["Sec-Ch-Ua-Model"] = `""`
	header["Sec-Ch-Ua-Platform-Version"] = `"19.0.0"`
	switch i {
	case 1: // html
		header["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
		header["Referer"] = "https://recu.me/"
		header["Sec-Fetch-Dest"] = "document"
		header["Sec-Fetch-Mode"] = "navigate"
		header["Sec-Fetch-Site"] = "none"
		header["Sec-Fetch-User"] = "?1"
		header["Upgrade-Insecure-Requests"] = "1"
	case 2: // playlist link
		header["Referer"] = videoUrl
		header["Sec-Fetch-Site"] = "same-origin"
		header["X-Requested-With"] = "XMLHttpRequest"
	default: // playlist
		header["Sec-Fetch-Site"] = "cross-site"
		delete(header, "Cookie")
		delete(header, "Sec-Ch-Ua-Full-Version-List")
		delete(header, "Sec-Ch-Ua-Arch")
		delete(header, "Sec-Ch-Ua-Bitness")
		delete(header, "Sec-Ch-Ua-Full-Version")
		delete(header, "Sec-Ch-Ua-Model")
		delete(header, "Sec-Ch-Ua-Platform-Version")
	}
	return
}
