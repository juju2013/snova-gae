package service

import (
	"appengine"
	"appengine/urlfetch"
	"event"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"util"
)

func buildHTTPRequest(ev *event.HTTPRequestEvent) *http.Request {
	req, err := http.NewRequest(ev.Method, ev.Url, &(ev.Content))
	if err != nil {
		return nil
	}
	var slen int = len(ev.Headers)
	for i := 0; i < slen; i++ {
		header := ev.Headers[i]
		req.Header.Add(header.Name, header.Value)
	}
	return req
}

func buildHTTPResponseEvent(res *http.Response) *event.HTTPResponseEvent {
	ev := new(event.HTTPResponseEvent)
	ev.Status = uint32(res.StatusCode)
	for key, values := range res.Header {
		for _, value := range values {
			ev.AddHeader(key, value)
		}
	}
	b := make([]byte, res.ContentLength)
	if res.ContentLength > 0 {
		res.Body.Read(b)
		ev.Content.Write(b)
	}
	return ev
}

func fillErrorResponse(ev *event.HTTPResponseEvent, cause string) {
	str := "You are not allowed to visit this site via proxy because %s."
	content := fmt.Sprintf(str, cause)
	ev.SetHeader("Content-Type", "text/plain")
	ev.SetHeader("Content-Length", strconv.Itoa(len(content)))
	ev.Content.WriteString(content)
}

func Fetch(context appengine.Context, ev *event.HTTPRequestEvent) event.Event {
	errorResponse := new(event.HTTPResponseEvent)
	if Cfg.IsMaster == 1 {
		fillErrorResponse(errorResponse, "Proxy service is no enable in snova master node.")
		return errorResponse
	}
	if isInBlacklist(ev.GetHeader("Host")) {
		fillErrorResponse(errorResponse, "Current site is in blacklist.")
		return errorResponse
	}
	req := buildHTTPRequest(ev)

	if req == nil {
		errorResponse.Status = 400
		fillErrorResponse(errorResponse, "Invalid fetch url:"+ev.Url)
		return errorResponse
	}
	var t urlfetch.Transport
	t.Context = context
	t.Deadline, _ = time.ParseDuration("10s")
	t.AllowInvalidServerCertificate = true
	//t := &transport
	//t := &urlfetch.Transport{context, 0, true}
	retryCount := Cfg.RetryFetchCount
	for retryCount > 0 {
		resp, err := t.RoundTrip(req)
		if err == nil {
			res := buildHTTPResponseEvent(resp)
			if res.Status == 302 {
				rangeHeader := req.Header.Get("Range")
				if len(rangeHeader) > 0 {
					res.AddHeader("X-Range", rangeHeader)
				}
			}
			return res
		}
		context.Errorf("Failed to fetch URL[%s] for reason:%v", ev.Url, err)
		retryCount--
		if strings.EqualFold(req.Method, "GET") && strings.Contains(err.Error(), "RESPONSE_TOO_LARGE") {
			rangeLimit := Cfg.RangeFetchLimit
			rangestart := 0
			rangeheader := req.Header.Get("Range")
			if len(rangeheader) > 0 {
				rangestart, _ = util.ParseRangeHeaderValue(rangeheader)
			}
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangestart, rangeLimit-1))
		}
		if strings.Contains(err.Error(), "RESPONSE_TOO_LARGE"){
			time.Sleep(1 * time.Second)
			return Fetch(context, ev)
		}
		
	}
	errorResponse.Status = 408
	fillErrorResponse(errorResponse, "Fetch timeout for url:"+ev.Url)
	return errorResponse

}
