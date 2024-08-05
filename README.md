# CloudKeeper
Run it as a background process to backup your files to s3



## Features:
1. You can set any local directory under watch.
2. As soon as any of the following change occurs, it creates an event.
    - create a file
    - Delete a file
    - Modify the content of file
    - Rename a file
    - Move a file/folder into another one
    - An empty folder won't be pushed
    - If a folder contains just one file and you delete it, the folder also gets removed(s3 handles it itself)
3. These events are stored in a channel, and are then consumed and based on the event, `filepath` and `action`(add or delete from s3) to be taken is stored in-memory
4. And every `10 minutes` these datas are flushed to the database for persistence. And once every `24hrs` (configurable), those files are pushed to s3.
5. I used [`bbolt`](https://github.com/etcd-io/bbolt) to persist the metadata till it gets flushed to s3. Why `bbolt`?
    - It is very simple to use(trust me, it is! 🫣)
    - It is a single user DB, no hassle, nothing, just `go get` it and you are good to go 😎
    - It is Go native key/value store.
    - It doesn't require a full database server such as Postgres or MySQL.
    - Lastly spilling the secret, It looked interesting as you can import it directly into your project and run within the application. so, wanted to give it a try 😜 and given our limited requirement, it fits the usecase.
  
## How to use?

1. Clone the repo
2. To install Dependencies use `go run` (or `go test` or `go build` for that matter) any external dependencies will automatically (and recursively) be [downloaded](https://www.alexedwards.net/blog/an-overview-of-go-tooling#build-and-deployment:~:text=when%20you%20use,the%20latest%20commit.)
3. Set the following environment variables in .env file(replace with your values, these are dummy ones for ref. :p) or use command like, `export BACKUP_INTERVAL=24`
   
```
BACKUP_DIR=/home/praveen/notifyTest
S3_BACKUP_INTERVAL=24
S3_BACKUP_INTERVAL_UNIT=hours           # one of hour(s)/minute(S)/second(s)
DB_PERSISTENCE_INTERVAL=10
DB_PERSISTENCE_INTERVAL_UNIT=minute      # one of hour(s)/minute(S)/second(s)
S3_BUCKET=backupbucket-praveen
S3_BUCKET_PREFIX=experimenting/
AWS_ACCESS_KEY=AKGHYUU67PraveenIsGood36tYUI
AWS_SECRET_ACCESS_KEY=Htyf5JED/E9EPraveenIsGoodwPRLhtyMh6jgdsFT
AWS_REGION=us-east-1
```

3. Build it: `go build -o anyName ./cmd/cloudkeeper`
4. Run: `./anyName`
5. Now go make changes and see for yourself.
6. you want a script to run as a daemon, in the background and never end‐
ing.

```
nohup ./upload >>/Home/tmp/log/cloudkeeper.log 2>&1 <&- &
```

> Note: Make sure you have your `/Home/tmp/log/cloudkeeper.log` created with necessary permisions. Here, `upload` is your binary executable obtained by running `go build` command, you can have any name.

7. Don't wan't to store the logs? Use

```
nohup ./upload 0<&-1>/dev/null 2>&1 &
```

Writing to `/dev/null` effectively throws away all the outputs from this program.

   
> If faced with any issue, raise an issue here(I promise, I will reply within seconds :xd.. Yes, I am the Flash🫣)



## Todo

 1. Write unit tests.
 2. DB transactions are not being hendled well.
 3. Better error-handling, will introduce a custom logger, I have made it a mess, looking where has the error actually occured is a nightmare rightnow.
 4. Imlement this project using checksum approach, instead of capturing every Fs event and benchmark them both.
 5. Work on notification part.

## Result

It is working as intended:
- Success message from terminal:
  
<img width="316" alt="image" src="https://github.com/user-attachments/assets/3ec58364-b656-400d-a842-e24507c7b01c">

- Data correctly reflecting in my s3 bucket:

<img width="657" alt="image" src="https://github.com/user-attachments/assets/13641e1d-f80f-40f8-ac3e-db50e89cea2e">


> Note: Here, I have used [`notify`](https://github.com/rjeczalik/notify) package, and not [fsnotify](https://github.com/fsnotify/fsnotify), because the later does not support recursive watching.