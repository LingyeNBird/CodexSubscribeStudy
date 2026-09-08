<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import type { Study } from "./types";
import AccountSurvey from "./AccountSurvey.vue";
import SurveyResultsPage from "./pages/SurveyResultsPage.vue";

const study = ref<Study | null>(null);
const route = ref(location.hash.slice(1) || "/");
const loading = ref(true);
const refreshing = ref(false);
const error = ref("");
const lastRefresh = ref("");
const controller = new AbortController();
let timer: ReturnType<typeof setInterval> | undefined;
const detail = computed(() => route.value === "/studies/gpt6-components");
const surveyRoute = computed(() =>
  ["/studies/chatgpt-account-survey", "/studies/chatgpt-account-survey/results"].includes(
    route.value,
  ),
);
const factorCause = ref("cache_read");
import HomePage from "./pages/HomePage.vue";
import StudyPage from "./pages/StudyPage.vue";
import MethodPage from "./pages/MethodPage.vue";
import PrivacyPage from "./pages/PrivacyPage.vue";
async function load() {
  if (refreshing.value) return;
  refreshing.value = true;
  const request = new AbortController();
  const abort = () => request.abort();
  controller.signal.addEventListener("abort", abort, { once: true });
  const timeout = setTimeout(abort, 15000);
  try {
    const response = await fetch("/api/studies/gpt6-components", {
      credentials: "omit",
      cache: "no-store",
      signal: request.signal,
    });
    if (!response.ok) throw new Error("service");
    const data: Study = await response.json();
    if (controller.signal.aborted) return;
    study.value = data;
    error.value = "";
    lastRefresh.value = new Date().toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    if (!controller.signal.aborted)
      error.value = "暂时无法读取科研统计。已有内容可能不是最新，请稍后重试。";
  } finally {
    clearTimeout(timeout);
    controller.signal.removeEventListener("abort", abort);
    loading.value = false;
    refreshing.value = false;
  }
}
function navigate() {
  route.value = location.hash.slice(1) || "/";
  window.scrollTo({ top: 0, behavior: "instant" });
}
onMounted(() => {
  window.addEventListener("hashchange", navigate);
  void load();
  timer = setInterval(() => {
    if (document.visibilityState === "visible") void load();
  }, 60000);
});
onBeforeUnmount(() => {
  controller.abort();
  if (timer) clearInterval(timer);
  window.removeEventListener("hashchange", navigate);
});
</script>

<template>
  <a class="skip" href="#main">跳到正文</a>
  <div class="site-shell">
    <header class="topbar">
      <a class="brand" href="#/" aria-label="共研首页"
        ><span class="brand-icon" aria-hidden="true"><i></i><i></i><i></i></span
        ><span>共研</span></a
      >
      <nav aria-label="主导航">
        <a href="#/" :aria-current="route === '/' || detail || surveyRoute ? 'page' : undefined"
          >研究项目</a
        ><a href="#/method" :aria-current="route === '/method' ? 'page' : undefined">研究方法</a
        ><a href="#/privacy" :aria-current="route === '/privacy' ? 'page' : undefined"
          >隐私与参与</a
        >
      </nav>
      <a
        class="source-link"
        href="https://github.com/LingyeNBird/CodexSubscribeStudy"
        target="_blank"
        rel="noreferrer"
        >开源代码 ↗</a
      >
    </header>
    <main id="main">
      <div v-if="error && (route === '/' || detail)" class="banner error" role="alert">
        <span>{{ error }}</span
        ><button @click="load" :disabled="refreshing">重新获取</button>
      </div>
      <HomePage v-if="route === '/'" :study="study" :loading="loading" />
      <template v-else-if="surveyRoute">
        <KeepAlive include="AccountSurvey">
          <AccountSurvey v-if="route === '/studies/chatgpt-account-survey'" />
          <SurveyResultsPage v-else />
        </KeepAlive>
      </template>
      <StudyPage
        v-else-if="detail"
        :study="study"
        :loading="loading"
        :refreshing="refreshing"
        :last-refresh="lastRefresh"
        v-model:factor-cause="factorCause"
        @refresh="load"
      />
      <MethodPage v-else-if="route === '/method'" />
      <PrivacyPage v-else-if="route === '/privacy'" />
      <section v-else class="empty-evidence">
        <h1>这个页面还不存在。</h1>
        <a class="button" href="#/">回到研究首页</a>
      </section>
    </main>
    <footer>
      <a class="footer-brand" href="#/">共研 <span>Codex Subscribe Study</span></a>
      <p>证据可以汇聚，不确定性不该被隐藏。</p>
      <a href="#/privacy">去标识化 · 长期保留</a>
    </footer>
  </div>
</template>

<style scoped src="./App.css"></style>
