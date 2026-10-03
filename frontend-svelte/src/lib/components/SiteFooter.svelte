<script lang="ts">
	import { config } from '#lib/config/index.ts';

	// 公安备案号里带中文（如「京公网安备 110105020xxxxx号」），查询链接只要数字部分
	function cleanChineseCharacters(str: string) {
		return str.replace(/[\u4e00-\u9fa5]/g, '');
	}
</script>

<!--
  页脚。图片一律用原生 <img> 并写死 width/height：
  - 属性声明固有比例，浏览器在图下载完之前就知道要占多大位置，避免整行先按 0 高度排一遍
    （Lighthouse 的 unsized-images 就是这条）；
  - 显示尺寸由固有值决定，窄屏由 site.css 的 `footer .one-line img { height:1.2em }` 接管。
  旧站这里踩过 el-image 的坑：它默认 lazy，SSR 只输出占位 div，HTML 里连 <img> 都没有。
-->
<footer>
	<div class="one-line">
		Copyright © nomdn &amp; IP 查询 2026 |
		<img src="/ipv6-s1.svg" alt="IPv6 相关标识" width="190" height="36" /> |
		<img src="/ssl-s1.svg" alt="SSL 相关标识" width="85" height="36" /> | All right reserved
	</div>
	<div class="one-line">
		{#if config.ICP}
			<a href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer">{config.ICP}</a>
			<span>&nbsp;|&nbsp;</span>
		{/if}
		{#if config.GongAn}
			<img
				src="/备案图标.png"
				alt=""
				width="36"
				height="40"
				style="height: 1em; width: 1em;"
			/>
			<a
				href={'https://beian.mps.gov.cn/#/query/webSearch?code=' +
					cleanChineseCharacters(config.GongAn)}
				target="_blank"
				rel="noreferrer">{config.GongAn}</a
			>
			<span>&nbsp;|&nbsp;</span>
		{/if}
		<a href="https://www.china-ipv6.cn/" target="_blank" rel="noreferrer">国家IPv6发展监测平台</a>
		&nbsp;|&nbsp;请遵守中国法律法规&nbsp;|&nbsp;
		<a href="https://github.com/nomdn/ipw-cn" target="_blank" rel="noreferrer">Github</a>&nbsp;|&nbsp;
		<a href="https://qm.qq.com/q/E1CGjkqgG6" target="_blank" rel="noreferrer">QQ用户交流群</a>
	</div>
	<div class="one-line">
		致力于普及IPv6，推进IPv6规模部署和应用，以全面推进IPv6技术创新与融合应用为主线，以提升应用广度深度为主攻方向
	</div>
</footer>
