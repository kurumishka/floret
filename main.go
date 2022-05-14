package main

import (
    "flag"
    "os"
    "fmt"
    "log"
    "io/ioutil"
    "encoding/json"
    "strings"
)

var (
    verbose   = flag.Bool("v", false, "подробные логи.")
    colorize  = flag.Bool("c", false, "расскрашивать картинки.")
    noCaps    = flag.Bool("no-caps", false, "не прикреплять описания.")

    token     = flag.String("token", "", "использовать заданный токен.")
    aid       = flag.String("aid", "", "использовать заданный album_id.")
    gid       = flag.String("gid", "", "использовать заданный group_id.")

    configDir = flag.String("conf", "./res/config.json", "файл конфига.")
    picsDir   = flag.String("pics", "./res/pics/", "директория с картинками.")
    capsDir   = flag.String("caps", "./res/captions.conf", "директория с описаниями.")

    threads   = flag.Uint64("th", 1, "кол-во потоков.")
    iters     = flag.Uint64("i", 1, "кол-во итерации.")
    timeout   = flag.Uint64("t", 0, "время ожидания (сек) между итерациями.")
)

var logging = func(extra bool, args ...interface{}) {
    if extra && !*verbose {
        return
    }
    log.Println(args...)
}

type Attachments struct {
    Pictures [][]byte
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
    caps, err := ioutil.ReadFile(capsPath)
    if err != nil {
        return nil, err
    }
    if !strings.HasSuffix(picsPath, "/") {
        picsPath += "/"
    }
    logging(true, "инициализируём байтовые массивы для картинок...")
    var (
        attachments = new(Attachments)
        failed      = 0
        predicate   = func(name string) bool {
                return strings.HasSuffix(name, ".jpg") ||
                       strings.HasSuffix(name, ".png") ||
                       strings.HasSuffix(name, ".jpeg")
                 }
    )
    for _, pic := range pics {
        if predicate(pic.Name()) {
            cont, err := ioutil.ReadFile(picsPath + pic.Name())
            if err != nil {
                logging(true, err)
                logging(true, "не удалось инициализировать", pic.Name())
                failed++
                continue
            }
            attachments.Pictures = append(attachments.Pictures, cont)
        }
    }
    logging(true, fmt.Sprintf("%d/%d картинок инициализировано.",
                              len(attachments.Pictures),
                              len(attachments.Pictures) + failed))
    attachments.Captions = strings.Split(string(caps), "\n\n")
    return attachments, nil
}

func parseConfig() *Config {
    config := Config{Credits: Credits{ *token, *gid, *aid }}
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
    logging(true, "ok, необходимые данные получены.\n", config)
    return &config
}

func main() {
    flag.Parse()
    config := parseConfig()
    logging(true, "проверяем валидность конфига...")
    ok := validateConfig(config)
    if !ok {
        logging(false, "фатальная ошибка, проверка конфига не удалась.")
        os.Exit(2)
    }
    logging(true, "ok, конфиг прошёл проверку.")
    logging(true, "получаем URL для загрузки картинок на сервер...")
    upserver, err := getUploadServer(config)
    if err != nil {
        logging(true, err)
        logging(false, "фатальная ошибка, не удалось получить URL для загрузки.")
        os.Exit(2)
    }
    logging(true, "ok, UPSERVER:", upserver)

    answer, err := uploadOnServer(config, upserver)
    if err != nil {
        logging(false, "не удалось загрузить картинки на сервер.")
        logging(true, err)
    }
    logging(true, answer)
}
