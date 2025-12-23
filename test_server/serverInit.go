package test_server

import (
	"encoding/json"
	"fmt"
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
	"github.com/wangshiben/QuicFrameWork/server"
	"io"
	"net/http"
)

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func SimpleGetServer() {
	httpServer := server.NewHttpServer(":8090")
	httpServer.AddHttpHandler("/", http.MethodGet, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		request := r.GetRequest()
		all, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Printf("error: %s", err.Error())
		}
		fmt.Printf("get:\n source:%s \n \t%v ; \n \tbody: %s", request.RemoteAddr, request, string(all))
		resp := &Response{200, "success"}
		marshal, err := json.Marshal(&resp)
		if err != nil {
			return
		}
		_, err = w.Write(marshal)
		if err != nil {
			fmt.Printf("errorInResp: %s", err.Error())
		}
	})
	httpServer.StartHttpSerer()
}

func SimplePostServer() {
	httpServer := server.NewHttpServer(":8090")
	httpServer.AddHttpHandler("/post", http.MethodPost, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		request := r.GetRequest()
		fmt.Printf("%s", "{\"\r\n}")
		all, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Printf("error: %s", err.Error())
		}
		bodyData := string(all)
		fmt.Printf("get:\n source:%s \n \t%v ; \n \tbody: %s \n", request.RemoteAddr, request, bodyData)
		resp := &Response{200, "success"}
		marshal, err := json.Marshal(&resp)
		if err != nil {
			return
		}
		_, err = w.Write(marshal)
		if err != nil {
			fmt.Printf("errorInResp: %s", err.Error())
		}
	})
	httpServer.StartHttpSerer()
}
