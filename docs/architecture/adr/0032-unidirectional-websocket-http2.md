# ADR-0032 Keep the unidirectional WebSocket HTTP/2 stream open

Accepted on 2026-11-24 for OSS-TRANS-02.

On HTTP/2 an upgraded WebSocket continues to use the HTTP response writer. Centrifugo v6.10.3 and the imported handler returned before the connection loop had finished using it. In isolated tests this ended the stream and terminated the server process.

Cedrazeka runs the connection loop within the HTTP/2 request handler and uses the request context for the client. HTTP/1.1 retains its existing connection lifecycle after hijacking the TCP socket. The behavior applies in both naming modes.

The regression test connects through the Cedrazeka unidirectional HTTP/2 endpoint and checks its initial client push. A second bidirectional connection must also succeed. The shared CS-TRANS-07 comparison covers a bidirectional connection to both servers. [Known differences](../../compatibility/known-differences.md)
