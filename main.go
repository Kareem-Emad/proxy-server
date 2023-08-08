package main

import (
	"context"
	"io"
	"log"
	"os"
	"sproxy/ratelimiter"
	"time"

	"github.com/things-go/go-socks5"
	"github.com/things-go/go-socks5/statute"
)

type AuthMidd struct {
	ratelimiter ratelimiter.RateLimiter
}

func (am AuthMidd) GetCode() uint8 { return statute.MethodUserPassAuth }

func (am AuthMidd) Authenticate(reader io.Reader, writer io.Writer, userAddr string) (*socks5.AuthContext, error) {
	if _, err := writer.Write([]byte{statute.VersionSocks5, statute.MethodUserPassAuth}); err != nil {
		return nil, err
	}

	nup, err := statute.ParseUserPassRequest(reader)
	if err != nil {
		return nil, err
	}

	userId := string(nup.User)

	// adhoc method to get request size
	// since socks5 doesn't provide a clear interface to get payload size of request or injecting middleware
	// if we want to track inbound/outboud requests sizes, we may need to modify the pkg to support our purposes
	// or move this logic prior to the proxy server
	shouldAllowRequest, err := am.ratelimiter.UpdateTrafficUsageForUser(userId, len(nup.Bytes()))
	if err != nil {
		return nil, err
	}

	// Verify the password
	if !shouldAllowRequest {
		if _, err := writer.Write([]byte{statute.UserPassAuthVersion, statute.AuthFailure}); err != nil {
			return nil, err
		}
		return nil, statute.ErrUserAuthFailed
	}

	if _, err := writer.Write([]byte{statute.UserPassAuthVersion, statute.AuthSuccess}); err != nil {
		return nil, err
	}

	return &socks5.AuthContext{
		Method: statute.MethodUserPassAuth,
		Payload: map[string]string{
			"username": string(nup.User),
			"password": string(nup.Pass),
		},
	}, nil
}

func printExamples(lg *socks5.Std) {

	lg.Logger.Println("unsupported arguments format")
	lg.Logger.Println("examples ./sproxy server")
	lg.Logger.Println("./sproxy fetchUserStats uid")
	lg.Logger.Println("./sproxy fetchGlobalStats")
}

func startServer(logger *socks5.Std, rl ratelimiter.RateLimiter) {
	logger.Println("running socks5 server ...")

	server := socks5.NewServer(
		socks5.WithLogger(logger),
		socks5.WithAuthMethods([]socks5.Authenticator{AuthMidd{
			ratelimiter: rl,
		}}),
	)

	ticker := time.NewTicker(15 * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				val, err := rl.FetchGlobalTrafficStats()
				if err != nil {
					logger.Fatalf("failed to reads global stats  | err %s \n", err.Error())
				}
				logger.Printf("[PROXY_SERVER] current global usage is %d", val)

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	if err := server.ListenAndServe("tcp", ":8001"); err != nil {
		panic(err)
	}

	quit <- true
}

func main() {
	logger := socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))

	if len(os.Args) < 2 {
		printExamples(logger)
		return
	}

	rl, err := ratelimiter.NewCacheRateLimiter(context.Background(), nil)
	if err != nil {
		logger.Fatalf("Failed to construct rate limiter | err : %s \n", err.Error())
	}

	if os.Args[1] == "fetchUserStats" && len(os.Args) == 3 {
		val, err := rl.FetchTrafficStatsForUser(os.Args[2])
		if err != nil {
			logger.Fatalf("failed to reads stats for user | err %s \n", err.Error())
		}

		logger.Logger.Printf("user stats till now: %d\n", val)
		return
	}
	if os.Args[1] == "fetchGlobalStats" {
		val, err := rl.FetchGlobalTrafficStats()
		if err != nil {
			logger.Fatalf("failed to reads global stats  | err %s \n", err.Error())
		}

		logger.Logger.Printf("global stats till now: %d\n", val)
		return
	}

	if os.Args[1] != "server" {
		printExamples(logger)
		return
	}

	startServer(logger, &rl)
}
