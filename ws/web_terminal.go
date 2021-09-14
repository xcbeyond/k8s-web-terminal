package ws

import (
	"encoding/json"
	"k8s-web-terminal/utils"
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

type streamHandler struct {
	wsConn      *Client
	resizeEvent chan remotecommand.TerminalSize
}

// message from web socket client.
type xtermMessage struct {
	// message type.
	// resize: resize the terminal's size,
	// input:  client input.
	MsgType string `json:"type"`
	// msgType = input
	Input string `json:"input"`
	// msgType = resize
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// Next is executor's callback function, used to adjust the size of terminal.
func (handler *streamHandler) Next() (size *remotecommand.TerminalSize) {
	ret := <-handler.resizeEvent
	size = &ret
	return
}

// Read is executor's callback function, used to read client's input.
func (handler *streamHandler) Read(p []byte) (size int, err error) {
	var (
		msg      *WsMessage
		xtermMsg xtermMessage
	)

	// read message from terminal.
	if msg, err = handler.wsConn.WsRead(); err != nil {
		return 0, err
	}

	if err = json.Unmarshal(msg.Data, &xtermMsg); err != nil {
		return 0, err
	}

	// resize the terminal's size.
	if xtermMsg.MsgType == "resize" {
		handler.resizeEvent <- remotecommand.TerminalSize{Width: xtermMsg.Cols, Height: xtermMsg.Rows}
	} else if xtermMsg.MsgType == "input" {
		size = len(xtermMsg.Input)
		copy(p, xtermMsg.Input)
	}

	return
}

// Write is executor's callback function, used to write client's ouput.
func (handler *streamHandler) Write(p []byte) (size int, err error) {
	copyData := make([]byte, len(p))
	copy(copyData, p)
	size = len(p)
	err = handler.wsConn.WsWrite(websocket.TextMessage, copyData)
	return
}

// WebSocketHandler is WebSocket handler.
func WebSocketHandler(resp http.ResponseWriter, req *http.Request) {
	var (
		wsConn   *Client
		executor remotecommand.Executor
		err      error
	)

	if err = req.ParseForm(); err != nil {
		return
	}
	namespace := req.Form.Get("namespace")
	podName := req.Form.Get("podName")
	containerName := req.Form.Get("containerName")

	if wsConn, err = InitWebsocket(resp, req); err != nil {
		log.Error(err)
		return
	}

	defer wsConn.WsClose()

	// k8s client connection from k8s config file.
	k8sClient := utils.Client{KubeConfigPath: "/Users/xcbeyond/.kube/config"}
	if err := k8sClient.NewClient(); err != nil {
		log.Error("Failed to create k8s client: ", err)
		return
	}

	// request url, for example:
	// http://172.18.11.25:8081/api/v1/namespaces/default/pods/reviews-v3-65495777bc-sh7tm/exec?command=sh&container=reviews&stderr=true&stdin=true&stdout=true&tty=true
	sshReq := k8sClient.K8sClient().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: containerName,
			Command:   []string{"bash"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	// creat container's connection.
	if executor, err = remotecommand.NewSPDYExecutor(k8sClient.RestConfig(), "POST", sshReq.URL()); err != nil {
		log.Error(err)
		return
	}

	// data stream callback
	handler := &streamHandler{wsConn: wsConn, resizeEvent: make(chan remotecommand.TerminalSize)}
	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               true,
	})
	if err != nil {
		log.Error(err)
		return
	}
}
