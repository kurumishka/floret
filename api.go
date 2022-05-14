package main

import (
    "encoding/json"
    "net/http"
    "net/url"
    "io/ioutil"
    "fmt"
    "mime/multipart"
    "bytes"
    "math/rand"
)

const (
    API_LINK = "https://api.vk.com/method/"
    VERSION  = "5.131"
)

type ErrorT struct {
    ErrorCode int `json:"error_code"`
    ErrorMsg string `json:"error_msg"`
}

type Error struct {
    ErrorV ErrorT `json:"error"`
}

func (e Error) String() string {
    return fmt.Sprintf("Error_code = %d: %s",
                       e.ErrorV.ErrorCode, e.ErrorV.ErrorMsg)
}

type GetUploadServerResponse struct {
    UploadUrl string `json:"upload_url"`
    AlbumId   int `json:"album_id"`
    UserId    int `json:"user_id"`
}

type GetUploadServer struct {
    GetUploadServerResponse `json:"response"`
}

type UploadServerResponse struct {
    Server     int
    PhotosList string `json:"photos_list"`
    Aid        int
    Hash       string
}

func callMethod(method string, params map[string]string) ([]byte, error) {
    query := url.Values{}
    for key, value := range params {
        query.Add(key, value)
    }
    link := API_LINK + method + "?" + query.Encode()
    logging(true, "GET:", link)
    resp, err := http.Get(link)
    if err != nil {
        return make([]byte, 0), err
    }
    defer resp.Body.Close()
    cont, err := ioutil.ReadAll(resp.Body)

    errInstance := new(Error)
    json.Unmarshal(cont, errInstance)
    if errInstance.ErrorV.ErrorCode != 0 {
        return make([]byte, 0), fmt.Errorf("%s", errInstance.String())
    }
    return cont, err
}

func validateConfig(config *Config) bool {
    params := map[string]string{
        "access_token": config.Token,
        "owner_id"    : fmt.Sprintf("-%s", config.Gid),
        "album_id"    : config.Aid,
        "count"       : "1",
        "v"           : VERSION,
    }
    _, err := callMethod("photos.get", params)
    if err != nil {
        logging(true, err)
        return false
    }
    return true
}

func getUploadServer(config *Config) (string, error) {
    params := map[string]string{
        "access_token": config.Token,
        "group_id"    : config.Gid,
        "album_id"    : config.Aid,
        "v"           : VERSION,
    }
    cont, err := callMethod("photos.getUploadServer", params)
    if err != nil {
        return "", err
    }
    upserver := new(GetUploadServer)
    json.Unmarshal(cont, upserver)
    if upserver.UploadUrl != "" {
        return upserver.UploadUrl, nil
    } else {
        return "", fmt.Errorf("getUploadServer: ошибка демаршализации.")
    }
}

func uploadOnServer(config *Config, upserver string) (*UploadServerResponse, error) {
    body := new(bytes.Buffer)
    writer := multipart.NewWriter(body)

    logging(true, "создаём multipart форму...")
    for i := 0; i < 5; i++ {
        file := config.Pictures[rand.Intn(len(config.Pictures))]
        fname := fmt.Sprintf("file%d", i)
        part, err := writer.CreateFormFile(fname, file.Name)
        if err != nil {
            writer.Close()
            return nil, err
        }
        part.Write(file.Content)
    }
    writer.Close()
    logging(true, "POST:", upserver)

    resp, err := http.Post(upserver, writer.FormDataContentType(), body)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    cont, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    response := new(UploadServerResponse)
    json.Unmarshal(cont, response)

    if response.PhotosList == "" {
        return nil, fmt.Errorf("сервер вернул неожиданный ответ: %s", cont)
    }
    return response, nil
}
