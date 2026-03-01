package grpc_server

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/sagernet/sing-box/log"
	"nekobox/grpc_server/gen"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
)

const (
	UrlTestStandard_RTT            = 0
	UrlTestStandard_Handshake      = 1
	UrlTestStandard_FisrtHandshake = 2
)

var errNoRedir = errors.New("no redir")

func getBetweenStr(str, start, end string) string {
	n := strings.Index(str, start)
	if n == -1 {
		n = 0
	}
	str = string([]byte(str)[n:])
	m := strings.Index(str, end)
	if m == -1 {
		m = len(str)
	}
	str = string([]byte(str)[:m])
	return str[len(start):]
}

func UrlTest(client *http.Client, link string, timeout int32, standard int) (int32, error) {
	if client == nil {
		return 0, fmt.Errorf("no client")
	}
	defer client.CloseIdleConnections()

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errNoRedir
	}

	var time_start time.Time
	var hsk_end time.Time
	var time_end time.Time
	var times int

	switch standard {
	case UrlTestStandard_FisrtHandshake:
		times = 1
	case UrlTestStandard_Handshake:
		times = 2
		rt := client.Transport.(*http.Transport)
		rt.DisableKeepAlives = true
	case UrlTestStandard_RTT:
		times = 2
	default:
		return 0, errors.New("unknown urltest standard")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		return 0, err
	}

	trace := &httptrace.ClientTrace{
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			hsk_end = time.Now()
		},
		GotFirstResponseByte: func() {
			time_end = time.Now()
		},
		WroteHeaders: func() {
			hsk_end = time.Now()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	for i := 0; i < times; i++ {
		time_start = time.Now()
		resp, err := client.Do(req)
		if err != nil {
			if errors.Is(err, errNoRedir) {
				err = nil
			} else {
				return 0, err
			}
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}

	if time_end.IsZero() {
		time_end = time.Now()
	}

	if standard == UrlTestStandard_RTT {
		time_start = hsk_end
	}

	return int32(time_end.Sub(time_start).Milliseconds()), nil
}

func TcpPing(address string, timeout int32) (ms int32, err error) {
	startTime := time.Now()
	c, err := net.DialTimeout("tcp", address, time.Duration(timeout)*time.Millisecond)
	endTime := time.Now()
	if err == nil {
		ms = int32(endTime.Sub(startTime).Milliseconds())
		c.Close()
	}
	return
}

func DoFullTest(ctx context.Context, in *gen.TestReq, core ProxyCore) (out *gen.TestResp, _ error) {
	out = &gen.TestResp{}
	httpClient := core.CreateProxyHttpClient()

	// Latency
	var latency string
	if in.FullLatency {
		t, _ := UrlTest(httpClient, in.Url, in.Timeout, UrlTestStandard_RTT)
		out.Ms = t
		if t > 0 {
			latency = fmt.Sprint(t, "ms")
		} else {
			latency = "Error"
		}
	}

	// UDP Latency
	var udpLatency string
	if in.FullUdpLatency {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		result := make(chan string)

		go func() {
			var startTime = time.Now()
			pc, err := core.DialContext(ctx, "udp", "8.8.8.8:53")
			if err == nil {
				defer pc.Close()
				dnsPacket, _ := hex.DecodeString("0000010000010000000000000377777706676f6f676c6503636f6d0000010001")
				_, err = pc.Write(dnsPacket)
				if err == nil {
					var buf [1400]byte
					_, err = pc.Read(buf[:])
				}
			}
			if err == nil {
				var endTime = time.Now()
				result <- fmt.Sprint(endTime.Sub(startTime).Abs().Milliseconds(), "ms")
			} else {
				log.Error("UDP Latency test error: ", err)
				result <- "Error"
			}
			close(result)
		}()

		select {
		case <-ctx.Done():
			udpLatency = "Timeout"
		case r := <-result:
			udpLatency = r
		}
		cancel()
	}

	// 入口 IP
	var in_ip string
	if in.FullInOut {
		_in_ip, err := net.ResolveIPAddr("ip", in.InAddress)
		if err == nil {
			in_ip = _in_ip.String()
		} else {
			in_ip = err.Error()
		}
	}

	// 出口 IP
	var out_ip string
	if in.FullInOut {
		resp, err := httpClient.Get("https://www.cloudflare.com/cdn-cgi/trace")
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			out_ip = getBetweenStr(string(b), "ip=", "\n")
			resp.Body.Close()
		} else {
			out_ip = "Error"
		}
	}

	// 下载
	var speed string
	if in.FullSpeed {
		if in.FullSpeedTimeout <= 0 {
			in.FullSpeedTimeout = 30
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(in.FullSpeedTimeout))
		result := make(chan string)
		var bodyClose io.Closer

		go func() {
			req, _ := http.NewRequestWithContext(ctx, "GET", in.FullSpeedUrl, nil)
			resp, err := httpClient.Do(req)
			if err == nil && resp != nil && resp.Body != nil {
				bodyClose = resp.Body
				defer resp.Body.Close()

				timeStart := time.Now()
				n, _ := io.Copy(io.Discard, resp.Body)
				timeEnd := time.Now()

				duration := math.Max(timeEnd.Sub(timeStart).Seconds(), 0.000001)
				resultSpeed := (float64(n) / duration) / MiB
				result <- fmt.Sprintf("%.2fMiB/s", resultSpeed)
			} else {
				result <- "Error"
			}
			close(result)
		}()

		select {
		case <-ctx.Done():
			speed = "Timeout"
		case s := <-result:
			speed = s
		}

		cancel()
		if bodyClose != nil {
			bodyClose.Close()
		}
	}

	fr := make([]string, 0)
	if latency != "" {
		fr = append(fr, fmt.Sprintf("Latency: %s", latency))
	}
	if udpLatency != "" {
		fr = append(fr, fmt.Sprintf("UDPLatency: %s", udpLatency))
	}
	if speed != "" {
		fr = append(fr, fmt.Sprintf("Speed: %s", speed))
	}
	if in_ip != "" {
		fr = append(fr, fmt.Sprintf("In: %s", in_ip))
	}
	if out_ip != "" {
		fr = append(fr, fmt.Sprintf("Out: %s", out_ip))
	}

	out.FullReport = strings.Join(fr, " / ")

	return
}
