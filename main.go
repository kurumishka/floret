package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "math/rand"
    "os"
    "strings"
    "time"
)

const (
    BUFSIZE = 2048
    ART     = `
      ________                __ 
     / ____/ /___  ________  / /_
    / /_  / / __ \/ ___/ _ \/ __/
   / __/ / / /_/ / /  /  __/ /_  
  /_/   /_/\____/_/   \___/\__/  
`
)

var (
    verbose  = flag.Bool("v", false, "подробные логи.")
    colorize = flag.Bool("c", false, "расскрашивать картинки. (не реализовано)")
    noCaps   = flag.Bool("no-caps", false, "не прикреплять описания.")

    token = flag.String("token", "", "использовать заданный токен.")
    aid   = flag.String("aid", "", "использовать заданный album_id.")
    gid   = flag.String("gid", "", "использовать заданный group_id.")

    configDir = flag.String("conf", "./res/config.json", "файл конфига.")
    picsDir   = flag.String("pics", "./res/pics/", "директория с картинками.")
    capsDir   = flag.String("caps", "./res/captions.conf", "директория с описаниями.")

    threads = flag.Uint64("th", 1, "кол-во потоков.")
    iters   = flag.Uint64("i", 1, "кол-во итерации.")
    timeout = flag.Uint64("t", 0, "время ожидания (сек) между итерациями.")
)

var logging = func(extra bool, args ...interface{}) {
    if extra && !*verbose {
        return
    }
    log.Println(args...)
}

type Picture struct {
    Content []byte
    Name    string
}

type Attachments struct {
    Pictures []Picture
    Captions []string
}

type Credits struct {
    Token string
    Gid   string `json:"group_id"`
    Aid   string `json:"album_id"`
}

type Config struct {
    Credits
    *Attachments
}

func (c Credits) String() string {
    return fmt.Sprintf("[%d_%d]", c.Gid, c.Aid)
}

func (c Config) String() string {
    return fmt.Sprintf("Token = %q; Gid = %q; Aid = %q; Pics = %d; Caps = %d.",
        c.Token, c.Gid, c.Aid, len(c.Pictures), len(c.Captions))
}

func parseConfigCredits(path string) (*Credits, error) {
    cont, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    credits := new(Credits)
    json.Unmarshal(cont, credits)
    return credits, nil
}

func parseConfigAttachments(picsPath, capsPath string) (*Attachments, error) {
    pics, err := os.ReadDir(picsPath)
    if err != nil {
        return nil, err
    }
    attachments := new(Attachments)
    if !*noCaps {
        caps, err := ioutil.ReadFile(capsPath)
        if err != nil {
            return nil, err
        }
        attachments.Captions = strings.Split(string(caps), "\n\n")
    }
    if !strings.HasSuffix(picsPath, "/") {
        picsPath += "/"
    }
    logging(false, "инициализируем байтовые массивы для картинок...")
    var (
        failed    = 0
        predicate = func(name string) bool {
            return strings.HasSuffix(name, ".jpg") ||
                strings.HasSuffix(name, ".png") ||
                strings.HasSuffix(name, ".jpeg")
        }
    )
    for _, pic := range pics {
        if predicate(pic.Name()) {
            fname := picsPath + pic.Name()
            cont, err := ioutil.ReadFile(fname)
            if err != nil {
                logging(true, err)
                logging(true, "не удалось инициализировать", pic.Name())
                failed++
                continue
            }
            attachments.Pictures = append(attachments.Pictures, Picture{cont, fname})
        }
    }
    logging(false, fmt.Sprintf("%d/%d картинок инициализировано.",
        len(attachments.Pictures),
        len(attachments.Pictures)+failed))
    return attachments, nil
}

func parseConfig() *Config {
    config := Config{Credits: Credits{*token, *gid, *aid}}
    logging(true, "читаем файл конфига...")
    credits, err := parseConfigCredits(*configDir)
    if err != nil {
        logging(true, err)
        logging(false, "ошибка, не удалось прочесть файл конфига.")
        logging(false, "будут насильно использованы значения флагов -t, -gid, -aid.")
    } else {
        config.Credits = *credits
    }
    logging(true, "получаем список картинок и описаний...")
    if config.Attachments, err = parseConfigAttachments(*picsDir, *capsDir); err != nil {
        logging(true, err)
        logging(false, "фатальная ошибка, не удалось получить необходимые данные.")
        os.Exit(2)
    }
    if len(config.Attachments.Pictures) == 0 {
        logging(false, "фатальная ошибка, не удалось найти ни одной картинки для загрузки.")
        os.Exit(2)
    }
    setConfigFlags(&config)
    logging(false, "ok, необходимые данные получены.")
    logging(true, config)
    return &config
}

func setConfigFlags(config *Config) {
    check := func(field *string, value *string) {
        if *value != "" {
            *field = *value
        }
    }
    check(&config.Token, token)
    check(&config.Gid, gid)
    check(&config.Aid, aid)
}

func upload(config *Config, upserver string, ch chan<- string) {
    start := time.Now()
    serverAnswer, err := uploadOnServer(config, upserver)
    if err != nil {
        ch <- fmt.Sprintf("%.2fs err, upload_on_server, %v",
            time.Since(start).Seconds(), err)
        return
    }
    params := map[string]string{
        "access_token": config.Token,
        "group_id":     config.Gid,
        "album_id":     config.Aid,
        "server":       fmt.Sprintf("%d", serverAnswer.Server),
        "photos_list":  serverAnswer.PhotosList,
        "hash":         serverAnswer.Hash,
        "v":            VERSION,
    }
    if !*noCaps {
        params["caption"] = config.Captions[rand.Intn(len(config.Captions))]
    }
    _, err = callMethod("photos.save", params)
    if err != nil {
        ch <- fmt.Sprintf("%.2fs err, photos.save, %v",
            time.Since(start).Seconds(), err)
    } else {
        ch <- fmt.Sprintf("%.2fs ok, картинки успешно сохранены.",
            time.Since(start).Seconds())
    }
}

func main() {
    flag.Parse()
    fmt.Println(ART)
    config := parseConfig()
    logging(false, "проверяем валидность конфига...")
    ok := validateConfig(config)
    if !ok {
        logging(false, "фатальная ошибка, проверка конфига не удалась.")
        os.Exit(2)
    }
    logging(false, "ok, конфиг прошёл проверку.")
    logging(true, "получаем URL для загрузки картинок на сервер...")
    upserver, err := getUploadServer(config)
    if err != nil {
        logging(false, err)
        logging(false, "фатальная ошибка, не удалось получить URL для загрузки.")
        os.Exit(2)
    }
    logging(true, "ok, UPSERVER:", upserver)

    answer, err := uploadOnServer(config, upserver)
    if err != nil {
        logging(false, "не удалось загрузить картинки на сервер.")
        logging(false, err)
    }
    logging(true, answer)
    logging(false, "сохраняем картинки в альбом...")
    logging(true, fmt.Sprintf("потоков = %d, итераций = %d.", *threads, *iters))

    ch := make(chan string, BUFSIZE)
    for i := uint64(1); i <= *iters; i++ {
        logging(false, fmt.Sprintf("итерация номер %d...", i))
        for j := uint64(0); j < *threads; j++ {
            go upload(config, upserver, ch)
        }
        logging(false, "потоки запущены.")
        for j := uint64(0); j < *threads; j++ {
            logging(false, <-ch)
        }
        logging(true, fmt.Sprintf("TIMEOUT: %d секунд.", *timeout))
        time.Sleep(time.Second * time.Duration(*timeout))
    }
    logging(false, "успешно завершено.")
}
