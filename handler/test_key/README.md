# AUTH TEST

> [!WARNING]
> This key pair just created for test
> DON NOT USE THEM IN PRACTICE

To generate new key, you can use `openssl`

```sh
openssl genpkey -algorithm rsa -out private.pem -pkeyopt bits:2048
openssl rsa -in private.pem -pubout -out public.pem
```
