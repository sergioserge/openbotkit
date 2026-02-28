package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const authPage = `<!DOCTYPE html>
<html><head><title>WhatsApp Login</title>
<script src="https://cdn.jsdelivr.net/npm/qrcodejs@1.0.0/qrcode.min.js"></script>
<style>body{font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f5f5f5}
.container{text-align:center;background:#fff;padding:2rem;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,.1)}
#status{margin-top:1rem;font-size:1.1rem}</style></head>
<body><div class="container"><h2>Scan QR Code with WhatsApp</h2>
<div id="qr"></div><p id="status">Loading...</p></div>
<script>
let qrEl=document.getElementById("qr"),statusEl=document.getElementById("status"),qrCode=null,hasQR=false;
function poll(){fetch("/api/qr").then(r=>r.json()).then(d=>{
if(d.authenticated){statusEl.textContent="Authenticated! You can close this tab.";statusEl.style.color="#16a34a";if(qrEl)qrEl.innerHTML="";return}
if(d.qr){hasQR=true;statusEl.textContent="Scan this QR code with your WhatsApp app";
if(!qrCode){qrCode=new QRCode(qrEl,{text:d.qr,width:256,height:256})}else{qrCode.clear();qrCode.makeCode(d.qr)}}
setTimeout(poll,hasQR?2000:5000)}).catch(()=>{statusEl.textContent="Connection lost";setTimeout(poll,5000)})}
poll();
</script></body></html>`

func ServeQR(ctx context.Context, client *Client, addr string) error {
	if addr == "" {
		addr = ":8085"
	}

	var mu sync.Mutex
	var currentQR string
	authenticated := false

	qrChan := make(chan string, 5)

	go func() {
		for qr := range qrChan {
			mu.Lock()
			currentQR = qr
			mu.Unlock()
		}
		mu.Lock()
		authenticated = true
		mu.Unlock()
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, authPage)
	})

	mux.HandleFunc("/api/qr", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		resp := map[string]any{
			"qr":            currentQR,
			"authenticated": authenticated,
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	server := &http.Server{Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	fmt.Printf("Open http://localhost:%d in your browser to scan the QR code\n", port)

	connectErr := client.ConnectWithQR(ctx, qrChan)

	// Keep serving briefly so the browser can poll and see the authenticated state.
	time.Sleep(5 * time.Second)
	server.Shutdown(context.Background())

	select {
	case err := <-errCh:
		return err
	default:
	}

	return connectErr
}
