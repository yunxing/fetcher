package main

import "net/http"
import "fmt"
import "time"
import "net"
import "regexp"
import "sync"
import "io/ioutil"

func dispatch_writer() chan string {
	c := make(chan string)
	go func() {
		for {
			name := <-c
			fmt.Println(name)
		}
	}()
	return c
}

var timeout = time.Duration(20 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

type TimeoutConn struct {
	conn    net.Conn
	timeout time.Duration
}

func NewTimeoutConn(conn net.Conn, timeout time.Duration) *TimeoutConn {
	return &TimeoutConn{
		conn:    conn,
		timeout: timeout,
	}
}

func (c *TimeoutConn) Read(b []byte) (n int, err error) {
	c.SetReadDeadline(time.Now().Add(c.timeout))
	return c.conn.Read(b)
}

func (c *TimeoutConn) Write(b []byte) (n int, err error) {
	c.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.conn.Write(b)
}

func (c *TimeoutConn) Close() error {
	return c.conn.Close()
}

func (c *TimeoutConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *TimeoutConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *TimeoutConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *TimeoutConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *TimeoutConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func main() {
	var wg sync.WaitGroup
	c := dispatch_writer()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		var index int = i
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				for k := 1; k < 100; k++ {
					client := &http.Client{
						Transport: &http.Transport{
							Dial: func(netw, addr string) (net.Conn, error) {
								conn, err := net.DialTimeout(netw, addr, time.Second*60)

								if err != nil {
									return nil, err
								}
								return NewTimeoutConn(conn, time.Second*60), nil
							},
							ResponseHeaderTimeout: time.Second * 60,
						},
					}
					url := fmt.Sprintf("https://calnet.berkeley.edu/directory/search.pl?search-type=uid&search-base=student&search-term=%02d%d%02d*&search=Search", k, index, j)

					resp, err := client.Get(url)
					if err != nil {
						fmt.Println(err)
						continue
					}
					defer resp.Body.Close()
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						fmt.Println(err)
						continue
					}
					r, err := regexp.Compile("class=\"underline\">(.*)</a></td>\n.*\n.*mailto\\:(.*@\\w+.\\w+)\"")
					if err != nil {
						fmt.Println(err)
						continue
					}
					all := r.FindAllStringSubmatch(string(body), -1)
					for _, p := range all {
						c <- fmt.Sprintf("%s | %s", p[1], p[2])
					}
				}
			}
		}()
	}
	wg.Wait()
}
