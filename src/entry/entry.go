package entry

import (
	"appengine"
	"appengine/user"
	//"appengine/xmpp"
	"bytes"
	//"encoding/base64"
	"event"
	"fmt"
	"misc"
	"net/http"
	"service"
	"strconv"
)

var serverInited bool = false

func init() {
	//event.InitEvents(new(handler.DispatchEventHandler))
	event.Init()
	http.HandleFunc("/", IndexEntry)
	http.HandleFunc("/admin", AdminEntry)
	http.HandleFunc("/invoke", HTTPEventDispatch)
	http.HandleFunc("/_ah/start", BackendInit)
	//warmup request is no available in GO runtime now
	http.HandleFunc("/_ah/warmup", InitGAEServer)
	//xmpp.Handle(XMPPEventDispatch)
}

func initGAEProxyServer(ctx appengine.Context) {
	if !serverInited {
		service.LoadServerConfig(ctx)
		if service.Cfg.IsMaster == 1 {
			service.InitMasterService(ctx)
		}
		ctx.Infof("InitGAEServer Invoked!")
		serverInited = true
	}
}

func InitGAEServer(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	initGAEProxyServer(ctx)
	w.WriteHeader(http.StatusOK)
}

const adminFrom = `
<html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
    <title>snova-gae(Go) V%s admin</title>
  </head>
  <body>
    <table width="800" border="0" align="center">
            <tr><td align="center">
                <b><h1>All Users:%v</h1></b>
            </td></tr>
             <tr><td align="center">
                <a href="%s">sign out</a>
            </td></tr>
    </table>
  </body>
</html>
`

const signoutFrom = `
<html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
    <title>snova-gae(Go) %s admin</title>
  </head>
  
   <body>
    <table width="800" border="0" align="center">
            <tr><td align="center">
                <p>Hello, %s! You are not the admin of this application, please 
<a href="%s">sign out</a> first, then login again.</p>
            </td></tr>
    </table>
    
  </body>
</html>
`

func AdminEntry(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}
	if !user.IsAdmin(c) {
		url, _ := user.LogoutURL(c, "/admin")
		fmt.Fprintf(w, signoutFrom, misc.Version, u.String(), url)
		return
	}
	url, _ := user.LogoutURL(c, "/")
	//root := service.GetUserWithName(c, "root")
	fmt.Fprintf(w, adminFrom, misc.Version, service.Cfg.AllUsers, url)
}

const indexForm = `
<html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
    <title>snova-gae(Go) %s</title>
  </head>

  <body>
    <table width="800" border="0" align="center">
            <tr><td align="center">
                <b><h1>snova-gae(Go) %s server is running!</h1></b>
            </td></tr>
            <tr><td align="center">
                <a href="/admin">admin</a>
            </td></tr>
    </table>
  </body>
</html>
`

func IndexEntry(w http.ResponseWriter, r *http.Request) {
	//ctx := appengine.NewContext(r)
	initGAEProxyServer(appengine.NewContext(r))
	fmt.Fprintf(w, indexForm, misc.Version, misc.Version)
}

func BackendInit(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type HTTPEventSendService struct {
	writer http.ResponseWriter
}

func (serv *HTTPEventSendService) GetMaxDataPackageSize() int {
	return -1
}

func (serv *HTTPEventSendService) Send(buf *bytes.Buffer) {
	headers := serv.writer.Header()
	headers.Add("Content-Type", "application/octet-stream")
	headers.Add("Content-Length", strconv.Itoa(buf.Len()))
	serv.writer.WriteHeader(http.StatusOK)
	serv.writer.Write(buf.Bytes())
}

func decodeEventWithTags(content []byte) (*event.EventHeaderTags, event.Event, error) {
	var tags event.EventHeaderTags
	buf := bytes.NewBuffer(content)
	if ok := tags.Decode(buf); !ok {
		return nil, nil, fmt.Errorf("Failed to decode event header tags")
	}
	err, res := event.DecodeEvent(buf)
	if nil != err {
		return nil, nil, err
	}
	res = event.ExtractEvent(res)
	return &tags, res, nil
}

func HTTPEventDispatch(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	initGAEProxyServer(ctx)
	buf := make([]byte, r.ContentLength)
	r.Body.Read(buf)
	serv := new(HTTPEventSendService)
	serv.writer = w
	tags, ev, err := decodeEventWithTags(buf)
	if nil == err {
		service.HandleEvent(tags, ev, ctx, serv)
		return
	}
	ctx.Errorf("Failed to parse HTTP event:" + err.Error())
}
