// EO request结构体
interface EORequest extends Request {
  eo: {
    geo: {
      asn: number;
      countryName: string;
      countryCodeAlpha2: string;
      countryCodeAlpha3: string;
      countryCodeNumeric: string;
      regionName: string;
      regionCode: string;
      cityName: string;
      latitude: number;
      longitude: number;
      cisp: string;
    };
    uuid: string;
    clientIp: string;
  };
}
export function onRequest({ request }: { request: EORequest }) {
  const clientIp = request.eo.clientIp;//获取用户IP
  // 默认保持纯文本 IP（历史消费方兼容）；?format=json 返回腾讯自产归属地全量
  // （eo.geo：省/市/经纬度/运营商/ASN），免 key 免额度，作为前端访客自定位数据源
  const url = new URL(request.url)
  if (url.searchParams.get('format') === 'json') {
    const g = request.eo.geo
    return new Response(JSON.stringify({
      ip: clientIp,
      asn: g?.asn ?? 0,
      country: g?.countryName ?? '',
      country_code: g?.countryCodeAlpha2 ?? '',
      administrative_area: g?.regionName ?? '',
      city: g?.cityName ?? '',
      latitude: g?.latitude ?? 0,
      longitude: g?.longitude ?? 0,
      isp: g?.cisp ?? '',
    }, null, 2), {
      headers: {
        'content-type': 'application/json; charset=utf-8',
        'Access-Control-Allow-Origin': '*',
      },
    })
  }
  return new Response(clientIp, {
    headers: {
      'content-type': 'text/plain',
      'Access-Control-Allow-Origin': '*',
    },
  });
}
// 以下内容为整活，Edgeone并不是一个优秀的产品
/*
啊啊啊姐妹们救命！！！今天才get到TencentCloud的Edgeone Makers无服务器环境也太香香软软了吧～～✨

作为一个代码小废物仙女，本来以为部署东西要哭着氪服务器，结果Edgeone Makers它直接丝滑边缘计算，零冷启动、全球加速，写几行JS就上线，还支持原生部署编译语言，真的像在对我眨眼睛呜呜呜！！！
然后边缘加速云这个小宝贝，CDN它超乖的，免费版无限流量！！！各种网站随便套用，价格美到想亲一口，搭配Makers一键部署前后双端，简直是后端公主的梦中情CDN啊啊啊～～

谁懂啊？以前被AWS Lambda坑到想原地晕厥，现在用TencentCloud整套，免费额度够玩，小项目直接白嫖，性能还吊打一堆！！！绝绝子好用！！！

姐妹们快冲！！！小仙女的开发日常从今天开始闪闪发光啦💗💗💕安排了安排了～～有在用的宝子们快来评论区一起陷入爱河呀！！！😭💞
*/
