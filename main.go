package main

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	emailFlag    = false
	telegramFlag = false
	sid          = "1155012345"
	pw           = "my password"
	email        = "my gmail@gmail.com"
	emailpw      = "my gmail pw"
	smtpHost     = "smtp.gmail.com"
	smtpAddr     = "smtp.gmail.com:587"
	bot_token    = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	chat_id      = "123456789"
)

const (
	mainURL     = "https://cusis.cuhk.edu.hk/psc/csprd/CUHK/PSFT_HR/c/SA_LEARNER_SERVICES.SSR_SSENRL_GRADE.GBL"
	loginURL    = "https://cusis.cuhk.edu.hk/psc/csprd/CUHK/PSFT_HR/c/SA_LEARNER_SERVICES.SSR_SSENRL_GRADE.GBL?cmd=login&languageCd=ENG"
	loginSlt    = ".psloginbutton"
	termSlt     = "#DERIVED_SSS_SCT_SSR_PB_GO"
	icsidSlt    = `input[name="ICSID"]`
	titleSlt    = ".PSLEVEL1GRID .PSHYPERLINK a"
	gradeSlt    = ".PSLEVEL1GRID .PABOLDTEXT"
	telegramURL = "https://api.telegram.org/bot"
)

type update struct {
	time  string
	grade string
}

var newGrades = list.New()

var grades = make(map[string]string)

var hongKong *time.Location

func login() error {
	log.Println("login")
	res, err := http.PostForm(loginURL, url.Values{
		"timezoneOffset": {"-480"},
		"userid":         {sid},
		"pwd":            {pw},
	})
	if err != nil {
		return err
	}
	if err := res.Body.Close(); err != nil {
		return err
	}
	return nil
}

func selectTerm(icsid, term string) (*http.Response, error) {
	return http.PostForm(mainURL, url.Values{
		"ICType":                 {"Panel"},
		"ICElementNum":           {"0"},
		"ICStateNum":             {"1"},
		"ICAction":               {"DERIVED_SSS_SCT_SSR_PB_GO"},
		"ICXPos":                 {"0"},
		"ICYPos":                 {"0"},
		"ICFocus":                {""},
		"ICSaveWarningFilter":    {"0"},
		"ICChanged":              {"-1"},
		"ICResubmit":             {"0"},
		"ICSID":                  {icsid},
		"#ICDataLang":            {"ENG"},
		"SSR_DUMMY_RECV1$sels$0": {term},
	})
}

func getGrade(icsid, term string) (string, error) {
	res, err := selectTerm(icsid, term)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	f := func (_ int, s *goquery.Selection) string {
		return strings.Replace(s.Text(), "\u00a0", "_", -1)
	}
	title := doc.Find(titleSlt).Map(f)
	grade := doc.Find(gradeSlt).Map(f)
	var s []string
	for i := 0; i < len(title) && i < len(grade); i++ {
		if grades[term + title[i]] != grade[i] {
			grades[term + title[i]] = grade[i]
			s = append(s, fmt.Sprintf("%s: %s\n", title[i], grade[i]))
		}
	}
	return strings.Join(s, ""), nil
}

func updateGrade(s string) error {
	const format = "02 Jan 2006 15:04:05"
	newGrades.PushBack(update{time.Now().In(hongKong).Format(format), s})
	log.Println("new grades:", s)
	if emailFlag {
		if err := mail(s); err != nil {
			return err
		}
	}
	if telegramFlag {
		if err := telegram(s); err != nil {
			return err
		}
	}
	return nil
}

func mail(s string) error {
	auth := smtp.PlainAuth("", email, emailpw, smtpHost)
	msg := []byte("To: " + email + "\r\n" +
		"Subject: CU CeotGrade\r\n" +
		"\r\n" +
		"Your grade:" + s + "\r\n")
	return smtp.SendMail(smtpAddr, auth, email, []string{email}, msg)
}

func telegram(s string) error {
	_, err := http.PostForm(telegramURL+bot_token+"/sendMessage", url.Values{
		"chat_id": {chat_id},
		"text":    {s},
	})
	return err
}

func run() error {
	res, err := http.Get(mainURL)
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return err
	}
	if doc.Find(loginSlt).Length() != 0 {
		return login()
	}
	if doc.Find(termSlt).Length() != 0 {
		icsid, exists := doc.Find(icsidSlt).Attr("value")
		if !exists {
			return errors.New("no icsid")
		}
		var ss []string
		terms := doc.Find(".PSLEVEL2GRIDWBO input").Length()
		for term := 0; term < terms; term++ {
			if s, err := getGrade(icsid, fmt.Sprint(term)); err == nil {
				ss = append(ss, s)
			} else {
				return err
			}
		}
		s := strings.Join(ss, "")
		if len(s) != 0 {
			return updateGrade(s)
		}
		return errors.New("no new grades")
	}
	html, err := doc.Html()
	if err != nil {
		return err
	}
	return errors.New(html)
}

func loop() {
	for {
		if err := run(); err != nil {
			log.Println("Error:", err)
		}
	}
}

func home(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "New Grades:")
	for e := newGrades.Back(); e != nil; e = e.Prev() {
		u := e.Value.(update)
		fmt.Fprintln(res, "Time:", u.time, "Grades:", u.grade)
	}
}

func main() {
	if location, err := time.LoadLocation("Asia/Hong_Kong"); err != nil {
		hongKong = location
	} else {
		hongKong = time.FixedZone("Asia/Hong_Kong", 8*3600)
	}
	var err error
	http.DefaultClient.Jar, err = cookiejar.New(nil)
	if err != nil {
		log.Println(err)
		return
	}
	go loop()
	http.HandleFunc("/", home)
	ip := os.Getenv("OPENSHIFT_GO_IP")
	if ip == "" {
		ip = "localhost"
	}
	port := os.Getenv("OPENSHIFT_GO_PORT")
	if port == "" {
		port = "8080"
	}
	bind := fmt.Sprintf("%s:%s", ip, port)
	fmt.Printf("listening on %s...\n", bind)
	if err := http.ListenAndServe(bind, nil); err != nil {
		panic(err)
	}
}
