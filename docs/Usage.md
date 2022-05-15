# Использование
Скачайте релиз последней версии [отсюда.](https://github.com/ketsushiri/floret/releases) Запуск осуществляется из командной строки с опциональным указанием флагов настройки.
```bash
$ floret [flags]
```

[Ссылка на получение токена.](https://oauth.vk.com/authorize?client_id=2685278&scope=1073737727&redirect_uri=https://oauth.vk.com/blank.html&display=page&response_type=token&revoke=1) Айди группы и альбома выглядит как 213304858_284082955, где до подчёркивания айди группы, а после - альбома.

Если вы хотите использовать дефолтные настройки, то просто заполните стандартный файл конфига `./res/config.json`, закиньте картинки в папку `./res/pics/` и опционально заполните `./res/captions.conf` описаниями. [Формат описаний.](https://github.com/ketsushiri/floret/blob/main/docs/Flags.md#описания)

Далее можно запускать без указания каких-либо дополнительных флагов.

## Windows
TODO: compile and write.
