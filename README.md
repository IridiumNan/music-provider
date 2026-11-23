# mp3-stream

## API

- `/list/text` GET

Response Body

```txt
music1.mp3
music2.mp3
music3.mp3
...
```

If no available music

```txt
No available music, contact music provider please
```

---

- `/list/json` GET

Response Body

```json
{
    "code": 200,
    "list": [
        "music1.mp3",
        "music2.mp3",
        ...
    ]
}
```

If no available music

```json
{
    "code": 404,
    "list": null
}
```

---

- `/music?name=music_name&percent=20` GET

Read music stream on body

You can also try this on your browser

---

## Test

```bash
# run this backend first (ensure music on the ./music_data dir)
./mc 127.0.0.1:8080 ./music_data
```

```bash
# test for all api

# get music list with text format
curl "http://localhost:8080/list/text"

# get music list with json format
curl "http://localhost:8080/list/json"

# get music stream with specific percent
# percent 80 for example
# NOTE: After the ?key1=value1&key2=value2
# support key is 
# 1. name for music file name
# 2. percent for stream percent
curl "http://localhost:8080/music?name=陈奕迅葡萄成熟时.mp3&percent=80"
```
