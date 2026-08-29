<script setup lang="ts">
import { onMounted } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();

onMounted(async () => {
  const urlParam = route.query.ip;
  // ip 参数缺失时不要拼出 ?ip=undefined
  await navigateTo(urlParam ? { path: '/location', query: { ip: urlParam } } : { path: '/location' });
});
</script>

<template>
  <div class="redirecting">正在跳转到 IP 归属地查询…</div>
</template>

<style scoped>
.redirecting {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
  color: #999;
}
</style>
