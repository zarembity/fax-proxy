# Fax proxy service

Прокси сервис для отправки факсов на FreeSwitch

### Run in development ###
```go run . --config-file=config/config.yml --upload-path=/tmp```

### Run in production ###
```./fax-servic . --config-file=config/config.yml --upload-path=/tmp```

### Проброс портов с тестового на локальный, на случай если нужно форму тестить с локальной машины по 127.0.0.1###
```ssh -L 8087:127.0.0.1:8087 root@```

### Тест через CURL

```
curl -X POST http://xx.xx.xx.xx:8087/send -H 'content-type: multipart/form-data' -F phone-number=062 -F uploadFile=@/Users/path/Desktop/fax-file.tiff
```