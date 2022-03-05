package main

import (
	"encoding/json"
	"log"
	"os"
	strconv "strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/valyala/fasthttp"
)

func checkCode(bodyString string, code string, user *discordgo.User, guild string, channel string, diff time.Duration) {

	var response Response
	err := json.Unmarshal([]byte(bodyString), &response)

	if err != nil {
		return
	}
	if strings.Contains(bodyString, "redeemed") {
		if settings.Nitro.Delay {
			logWithTime("<yellow>[-] " + response.Message + "</> Delay: " + strconv.FormatInt(int64(diff/time.Millisecond), 10) + "ms")
		} else {
			logWithTime("<yellow>[-] " + response.Message + "</>")
		}
		webhookNitro(code, user, guild, channel, 0, response.Message)
	} else if strings.Contains(bodyString, "nitro") {
		f, err := os.Open("sound.mp3")
		if err != nil {
			log.Fatal(err)
		}

		var format beep.Format
		sound, format, err := mp3.Decode(f)
		if err != nil {
			log.Fatal(err)
		}
		defer sound.Close()

		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

		done := make(chan bool)
		speaker.Play(beep.Seq(sound, beep.Callback(func() {
			done <- true
		})))

		<-done
		nitroType := ""
		if reNitroType.Match([]byte(bodyString)) {
			nitroType = reNitroType.FindStringSubmatch(bodyString)[1]
		}

		if settings.Nitro.Delay {
			logWithTime("<green>[+] Nitro applied : </><cyan>" + nitroType + "</> Delay:" + strconv.FormatInt(int64(diff/time.Millisecond), 10) + "ms")
		} else {
			logWithTime("<green>[+] Nitro applied : </><cyan>" + nitroType + "</>")
		}
		webhookNitro(code, user, guild, channel, 1, nitroType)
		NitroSniped++
		if NitroSniped >= settings.Nitro.Max {
			SniperRunning = false
			time.AfterFunc(time.Hour*time.Duration(settings.Nitro.Cooldown), timerEnd)
			logWithTime("<yellow>[+] Stopping Nitro sniping for now</>")
		}
	} else if strings.Contains(bodyString, "Unknown Gift Code") {
		if settings.Nitro.Delay {
			logWithTime("<red>[x] " + response.Message + "</> Delay: " + strconv.FormatInt(int64(diff/time.Millisecond), 10) + "ms")
		} else {
			logWithTime("<red>[x] " + response.Message + "</>")
		}
	} else {
		logWithTime("<yellow>[?] " + response.Message + "</>")
		if settings.Nitro.Delay {
			logWithTime("<yellow>[?] " + response.Message + "</> Delay: " + strconv.FormatInt(int64(diff/time.Millisecond), 10) + "ms")
		} else {
			logWithTime("<yellow>[?] " + response.Message + "</>")
		}
		webhookNitro(code, user, guild, channel, -1, response.Message)
	}
	cache.Set(code, "", 1)

}

func getCookieString() {
	println("Cookies get requested")
	var strRequestURI = []byte("https://discord.com/")
	req := fasthttp.AcquireRequest()
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("accept-Encoding",  "gzip, deflate, br")
	req.Header.Set("accept-Language", "de,en-US;q=0.7,en;q=0.3")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("DNT", "1")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:97.0) Gecko/20100101 Firefox/97.0")
	req.Header.SetMethodBytes([]byte("GET"))
	req.SetRequestURIBytes(strRequestURI)
	res := fasthttp.AcquireResponse()

	if err := fasthttp.Do(req, res); err != nil {
		return
	}
	end := time.Now()
	diff := end.Sub(start)

	fasthttp.ReleaseRequest(req)

	body := res.Body()

	bodyString := string(body)
	fasthttp.ReleaseResponse(res)

	if res.Cookies() == nil {
		return ""
	}
	var cookies string
	for _, cookie := range res.Cookies() {
		cookies = cookies + cookie.Name + "=" + cookie.Value + "; "
	}
	cookies = cookies + "locale=en-US"
	println(cookies)
	return cookies
}

func checkGiftLink(s *discordgo.Session, m *discordgo.MessageCreate, link string, start time.Time) {

	code := reGiftLink.FindStringSubmatch(link)

	if len(code) < 2 {
		return
	}

	if len(code[2]) < 16 {
		logWithTime("<red>[=] Auto-detected a fake code: " + code[2] + " from " + m.Author.String() + "</>")
		return
	}

	_, found := cache.Get(code[2])
	if found {
		logWithTime("<red>[=] Auto-detected a duplicate code: " + code[2] + " from " + m.Author.String() + "</>")
		return
	}
	println("Checking Link...")
	var strRequestURI = []byte("https://discordapp.com/api/v8/entitlements/gift-codes/" + code[2] + "/redeem")
	req := fasthttp.AcquireRequest()
	req.Header.SetContentType("application/json")
	req.Header.Set("authorization", settings.Tokens.Main)
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-Encoding",  "gzip, deflate, br")
	req.Header.Set("accept-Language", "de,en-US;q=0.7,en;q=0.3")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("cookies", getCookieString())
	req.Header.Set("DNT", "1")
	req.Header.Set("Host", "discord.com")
	req.Header.Set("Referer", "https://discord.com/gifts/" + code[2])
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("TE", "trailers")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:97.0) Gecko/20100101 Firefox/97.0")
	req.Header.Set("X-Debug-Options", "bugReporterEnabled")
	req.Header.Set("X-Discord-Locale", "en-US")
	req.Header.Set("X-Super-Properties", "eyJvcyI6IldpbmRvd3MiLCJicm93c2VyIjoiRmlyZWZveCIsImRldmljZSI6IiIsInN5c3RlbV9sb2NhbGUiOiJkZSIsImJyb3dzZXJfdXNlcl9hZ2VudCI6Ik1vemlsbGEvNS4wIChXaW5kb3dzIE5UIDEwLjA7IFdpbjY0OyB4NjQ7IHJ2Ojk3LjApIEdlY2tvLzIwMTAwMTAxIEZpcmVmb3gvOTcuMCIsImJyb3dzZXJfdmVyc2lvbiI6Ijk3LjAiLCJvc192ZXJzaW9uIjoiMTAiLCJyZWZlcnJlciI6IiIsInJlZmVycmluZ19kb21haW4iOiIiLCJyZWZlcnJlcl9jdXJyZW50IjoiIiwicmVmZXJyaW5nX2RvbWFpbl9jdXJyZW50IjoiIiwicmVsZWFzZV9jaGFubmVsIjoic3RhYmxlIiwiY2xpZW50X2J1aWxkX251bWJlciI6MTE3MzAwLCJjbGllbnRfZXZlbnRfc291cmNlIjpudWxsfQ==)")
	
	var channelId = "null"
	if s.Token == settings.Tokens.Main {
		channelId = m.ChannelID
	}
	req.SetBody([]byte(`{"channel_id":` + channelId + `,"payment_source_id": ` + paymentSourceID + `}`))
	req.Header.SetMethodBytes([]byte("POST"))
	req.SetRequestURIBytes(strRequestURI)
	res := fasthttp.AcquireResponse()

	if err := fasthttp.Do(req, res); err != nil {
		return
	}
	end := time.Now()
	diff := end.Sub(start)

	fasthttp.ReleaseRequest(req)

	body := res.Body()

	bodyString := string(body)
	fasthttp.ReleaseResponse(res)

	guild, err := s.State.Guild(m.GuildID)
	if err != nil || guild == nil {
		guild, err = s.Guild(m.GuildID)
		if err != nil {
			println()
			checkCode(bodyString, code[2], s.State.User, "DM", m.Author.Username+"#"+m.Author.Discriminator, diff)
			return
		}
	}

	channel, err := s.State.Channel(m.ChannelID)
	if err != nil || guild == nil {
		channel, err = s.Channel(m.ChannelID)
		if err != nil {
			println()
			checkCode(bodyString, code[2], s.State.User, guild.Name, m.Author.Username+"#"+m.Author.Discriminator, diff)
			return
		}
	}

	logWithTime("<green>[-] " + s.State.User.Username + " sniped code: </><red>" + code[2] + "</> from  <magenta>[" + guild.Name + " > " + channel.Name + "]</>")

	checkCode(bodyString, code[2], s.State.User, guild.Name, channel.Name, diff)
}
