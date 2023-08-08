
## How to use

To run socks5 server
```
go run main.go server
```

To fetch traffic stats for certain user
```
go run main.go fetchUserStats user_id
``` 

To fetch traffic stats for global
```
go run main.go fetchGlobalStats
```


sample requeat to server
```
curl  --socks5  socks5://fierceRabbit:secretThicket@localhost:8001 http://www.google.com/
```

## Testing and coverage

To run tests
```
go test --coverpkg=./...  -coverprofile cover.out ./...
```

*note that some tests have dependency over redis-server being present on your machine and running, you can skip integration tests if it's not avaiable

To regenerate coverage report
```
go tool cover -html cover.out -o cover.html
```


## Techincal Flow

The diagram below shows how communication works once a request reaches proxy server, basically we have:
- socks5 recieving requests and with added middleware that tracks and checks user usage limits
- the package uses user id to record new request made in redis and also checks if user is over limit or not
- once new usage is recorded, it's checked against limits
- if the user went over limit or exactly at limit, then the user is marked as blocked 
- in future requests, the user won't be allowed to make any more proxied requests as he is blocked
- this may allow users to spill over their limit in the last few concurrent requests 


### syncing data

- the choice to put caches upfront and make database just a secondary source of truth storage is meant to
  - Give us more flexiblity in scaling, since caches are easily scalable and fast, unlike databases
  - Avoid scaling bottelnecks such as having to scale mysql instance or searching for other scaling solutions
  - leverage memory caches for small chunks of data to maitain minimal delays for response time as users are not expected to increase expoenentially, but their requests are  (would be great if users increase expoententially too )
  - database syncing happens periodically and at a constant pace that the database can handle

![diagram](./assets/img.png)  


## code structure

The package follows a simple structure where it's divided into two interface levels to allow flexiblity in changing implementation:
- Rate Limiter: Generic interface to implement rate limiting
- Cache: Generic interface to implement a cache layer using any caching mechanism and could be used with any rate limiter if needed
  
*Syncing between caches and persistant database is not included since it's out of the package scope and could be implemented in a better way utilizing sync mechanisms avaiable for the choic of DB.
![diagram](./assets/code.png)  
