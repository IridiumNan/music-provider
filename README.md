# mp3-stream

## Challenge

This website use challenge to authorize your identity.

Only this usage document and challenge endpoint can visit directly.

For visit other endpoints. You must use process as below.

1. Send a request to `/auth/challenge` then read nonce from response body.
2. Hash this nonce then sign with you rsa private key.
3. Create a **new request with body**

The `body` contains three sections:

1. original nonce you received
2. `\n`
3. the signature your generated based on hash value and your private key (base64 encoding)

---

## API

For all API below, you should provide your signature, see challenge section

- `/list/text` POST

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

- `/list/json` POST

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

- `/music?name=music_name&percent=20` POST

Read music stream on body
