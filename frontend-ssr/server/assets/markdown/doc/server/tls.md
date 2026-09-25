# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> 网站如何开启 TLS

## TLS是什么？

[RFC 8446: The Transport Layer Security (TLS) Protocol Version 1.3](https://datatracker.ietf.org/doc/html/rfc8446)

TLS allows client/server applications to communicate over the Internet in a way that is designed to prevent eavesdropping, tampering, and message forgery.

### Major Differences from TLS 1.2

## Wireshark 抓包分析

### TCP 三次握手

- SYN ![SYN](/doc/tls_tcp_01_syn.png)
- SYN ACK ![SYN ACK](/doc/tls_tcp_02_syn_ack.png)
- ACK ![ACK](/doc/tls_tcp_03_ack.png)

### TLS 协商

以 TLS 1.3 为例

- Client Hello ![Client Hello](/doc/tls_01_client_hello.png)
- Server Hello, Change Cipher Spec, Encrypted Extensions ![Server Hello, Change Cipher Spec, Encrypted Extensions](/doc/tls_02_server_hello_change_cipher_spec_encrypted_extensions.png)
- Certificate, Certificate Verify, Finished ![Certificate, Certificate Verify, Finished](/doc/tls_03_certificate_verify_finished.png)
- Change Cipher Spec, Finished ![Change Cipher Spec, Finished](/doc/tls_04_change_cipher_spec_finished.png)
