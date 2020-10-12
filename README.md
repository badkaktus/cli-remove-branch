A simple CLI-script that automatically delete open branch in gitlab.
Script check all opened branches and if branch start from numbers 
(for example `123-example-branch`) check issue status (issue #123). 
If issue was closed, that branch will delete.

It is also possible to send notifications to Rocket.Chat.

Available arguments:
```
glurl - Gitlab URL
gltoken - Gitlab Token 
glproject - Project ID in Gitlab 
rurl - Rocket.Chat URL
ruser - rocketchat username
rpass - password of rocketuser
rch - rocketchat channel to notify
```

Example:

```
go build -o deletebranch main.go
./deletebranch -glurl gitlab.dev -gltoken aaaaBBBBcccc1111 -glproject 1
```

Example with Rocket.Chat

```
go build -o deletebranch main.go
./deletebranch -glurl gitlab.dev -gltoken aaaaBBBBcccc1111 -glproject 1 -rurl https://rocket.company.io -ruser bot -rpass botpass -rch rocketchannel
```

Thats all.

PS If you need notify in another messenger, open issue.