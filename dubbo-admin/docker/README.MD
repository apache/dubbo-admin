Execute the command at folder of dubbo-admin

```Bash
docker-compose -f docker/dubbo.yml up -d
```
Then visit the http://localhost:8090 and login with root:root or guest:guest.


Stop the container by docker-compose down:

```Bash
docker-compose -f docker/dubbo.yml down
```