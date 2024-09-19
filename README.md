# cryDrv

## encrypted CRUD everything web drive

- data encryption on rest
- CRUD REST API
- arbitrary accounts via HTTP Basic Auth
- client can read and write all filepaths for the current account
- user can build custom DIY webpages
- serve index.html for directories

## Protocol

1. Server: starts with env var `secret_key` being a (base64 encoded) 32-octets random key
2. Client: sends HTTP request. No basic auth or cookie provided => must authenticate
---
3. Client: sends basic auth with arbitrary `username` and `password`
4. Server: `userSalt = hkdf(secret_key, salt=username)`
5. Server: `userKey = argon2id(password, salt=userSalt)`
6. Server: attach Cookie with value `userKey` to Client
---
7. Client: GET file at `path` "/a/b.c"
8. Server: `filename = hkdf(userKey, salt=userSalt + path)`, check `path` is not empty
9. Server: Serve file `filename` under webpath `path` (if it exists in filesystem)
10. Client: POST/PUT file `content` at `path` "/a/b.c"
11. Server: calculates `filename`, encrypts the file `file = aes256gcm(content, userKey, nonce)` and stores `file` under this path. `file` is encrypted chunkwise with a new `nonce` every 4MiB (plus PKCS#7 padding)
12. Client: DELETE file at `path` "/a/b.c"
13. Server: calculate `filename` and delete the file if it exists under this path
---
14. Client: uses the Cookie (see 6.) in addition to basic auth
15. Server: takes `username` from basic auth and `userKey` from cookie (remember `filename` is constructed using both `username` and `userKey`)
---
16. Server-Admin: closes the registration
17. Client: sends request with either (`username`, `password`) or (`username`, `userKey`)
18. Server: `userFingerprint = hkdf(userKey, salt=userSalt)`
19. Server: check `userFingerprint` is on allowlist. if not, print (`username`, `userFingerprint`) to server log. the server-admin can then add `userFingerprint` to the allowlist

## Threat model

- User has to trust the webserver blindly (as with all web apps)
- Webserver doesn't have to trust the storage/backup provider (e.g. cloud)
- Storage provider can still see file count, sizes and metadata => can guess possible file content types by size and track general activities via timestamps
- add HTTPS for transport encryption

## Setup

```yaml
services:
  crydrv:
    image: ghcr.io/knrdl/crydrv:edge
    restart: always
    environment:
      - OPEN_REGISTRATION=true  # default: false
      - MIN_PASSWORD_LENGTH=16  # default: 16
    # - SECRET_KEY=...  # generated on first start
    ports:
      - 8000:8000
    volumes:
      - ./data:/www  # chown -R 1000:1000 ./data

    mem_limit: 4g
    memswap_limit: 4g
```

getting started: see [webutils](./webutils)