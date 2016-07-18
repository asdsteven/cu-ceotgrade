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
	"time"
)

const (
	sid      = "1155012345"
	pw       = "my password"
	email    = "my gmail@gmail.com"
	emailpw  = "my gmail pw"
	smtpHost = "smtp.gmail.com"
	smtpAddr = "smtp.gmail.com:587"
)

const (
	mainURL  = "https://cusis.cuhk.edu.hk/psc/csprd/CUHK/PSFT_HR/c/SA_LEARNER_SERVICES.SSR_SSENRL_GRADE.GBL"
	loginURL = "https://cusis.cuhk.edu.hk/psc/csprd/CUHK/PSFT_HR/c/SA_LEARNER_SERVICES.SSR_SSENRL_GRADE.GBL?cmd=login&languageCd=ENG"
	loginSlt = ".psloginbutton"
	termSlt  = "#DERIVED_SSS_SCT_SSR_PB_GO"
	icsidSlt = `input[name="ICSID"]`
	gradeSlt = ".PABOLDTEXT"
)

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

func selectTerm(icsid string) error {
	res, err := http.PostForm(mainURL, url.Values{
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
		"SSR_DUMMY_RECV1$sels$0": {"0"},
	})
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromResponse(res)
	e := doc.Find(gradeSlt)
	if e.Length() != 0 {
		return updateGrade(e.Text())
	}
	return errors.New("no grade after term")
}

type update struct {
	time  string
	grade string
}

var newGrades = list.New()

var grades = list.New()

var hongKong *time.Location

func updateGrade(s string) error {
	const format = "02 Jan 2006 15:04:05"
	if grades.Back() != nil && grades.Back().Value.(update).grade == s {
		grades.PushBack(update{time.Now().In(hongKong).Format(format), s})
		if grades.Len() > 60 {
			grades.Remove(grades.Front())
		}
		return nil
	}
	grades.PushBack(update{time.Now().In(hongKong).Format(format), s})
	if grades.Len() > 20 {
		grades.Remove(grades.Front())
	}
	newGrades.PushBack(update{time.Now().In(hongKong).Format(format), s})
	log.Println("new grade:", s)
	return mail(s)
}

func mail(s string) error {
	auth := smtp.PlainAuth("", email, emailpw, smtpHost)
	msg := []byte("To: " + email + "\r\n" +
		"Subject: CU CeotGrade\r\n" +
		"\r\n" +
		"Your grade:" + s + "\r\n")
	return smtp.SendMail(smtpAddr, auth, email, []string{email}, msg)
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
		return selectTerm(icsid)

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
	fmt.Fprintln(res, "\nUpdates:")
	for e := grades.Back(); e != nil; e = e.Prev() {
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
