# <span style="background-color: #b95442;color: white;font-size: 0.43em;border-radius: 5px;padding: 2px 5px;">转载</span> tcpdump 分析 IPv6 包


```
# tcpdump icmp6 -XX
tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
listening on eth0, link-type EN10MB (Ethernet), capture size 262144 bytes
21:35:46.919301 IP6 240e:3b7:******** > guangzhou-xx-xx-xx-247: ICMP6, echo request, seq 5412, length 16
        0x0000:  5254 005f 7111 feee 8510 4bc5 86dd 6689  RT._q.....K...f.
        0x0010:  0400 0010 3a34 240e 0000 0000 0000 0000  ....:4$...2$n.-.
        0x0020:  e431 a261 4cd2 2402 4e00 1013 e500 0000  .1.aL.$.N.......
        0x0030:  940c 7377 7cbc 8000 5f66 e880 1524 61bf  ..sw|..._f...$a.
        0x0040:  3532 000c d76c                           52...l
```


```
# tcpdump ip6 proto 6
tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
listening on eth0, link-type EN10MB (Ethernet), capture size 262144 bytes
21:35:57.681602 IP6 beijing-xx-xx-xx-5.55998 > 2402:4e00:40:40::2:331.https: Flags [P.], seq 3521054660:3521054788, ack 3373296701, win 73, length 128
21:35:57.715035 IP6 2402:4e00:40:40::2:331.https > beijing-xx-xx-xx-5.55998: Flags [.], ack 128, win 64, length 0
21:35:57.718536 IP6 2402:4e00:40:40::2:331.https > beijing-xx-xx-xx-5.55998: Flags [P.], seq 1:616, ack 128, win 64, length 615
21:35:57.718553 IP6 beijing-xx-xx-xx-5.55998 > 2402:4e00:40:40::2:331.https: Flags [.], ack 616, win 79, length 0
21:35:59.079661 IP6 beijing-xx-xx-xx-5.55998 > 2402:4e00:40:40::2:331.https: Flags [P.], seq 128:166, ack 616, win 79, length 38
```


